// Package proxy is Tennon's data plane and MCP surface. Every connector is
// reachable at /mcp/<name> over streamable-HTTP. On each request the proxy
// authenticates the caller, authorizes the call, renders the connector's
// credential in-process, and forwards to the upstream MCP server. The caller
// never sees the credential.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/specconn"
)

type ctxKey int

const subjectKey ctxKey = iota

// SubjectFromContext returns the authenticated subject stored by the proxy's
// auth middleware, or the anonymous subject if none is present.
func SubjectFromContext(ctx context.Context) domain.Subject {
	if s, ok := ctx.Value(subjectKey).(domain.Subject); ok {
		return s
	}
	return domain.Anonymous()
}

// Proxy holds the live routing table of connector proxies and serves them.
type Proxy struct {
	renderer domain.SecretRenderer
	runtime  domain.Runtime
	authz    domain.Authorizer
	verifier domain.IdentityVerifier
	store    domain.Store
	log      *slog.Logger

	mu        sync.RWMutex
	conns     map[string]*connProxy
	aggregate *aggregateServer // cached aggregate endpoint
	pending   map[string]chan *mcp.CallToolResult // execution id → result channel
}

// aggregateServer wraps the aggregate MCP server and its handler.
type aggregateServer struct {
	server  *mcp.Server
	handler http.Handler
}

// New builds a Proxy. The concrete ports are supplied by the composition root.
func New(renderer domain.SecretRenderer, rt domain.Runtime, authz domain.Authorizer, verifier domain.IdentityVerifier, store domain.Store, log *slog.Logger) *Proxy {
	if log == nil {
		log = slog.Default()
	}
	return &Proxy{
		renderer: renderer,
		runtime:  rt,
		authz:    authz,
		verifier: verifier,
		store:    store,
		log:      log,
		conns:    map[string]*connProxy{},
		pending:  map[string]chan *mcp.CallToolResult{},
	}
}

// Register makes a connector reachable: it renders the credential, ensures the
// upstream runtime, dials it, discovers tools, and installs the route. It
// returns the discovered tools so the controller can update status.
//
// For spec connectors (OpenApiConnector/GraphqlConnector), it compiles tools
// in-process using specconn and registers them via RegisterServer instead of
// dialing an upstream.
func (p *Proxy) Register(ctx context.Context, c *domain.Connector) ([]domain.Tool, error) {
	// Handle spec connectors: compile tools in-process.
	if c.Kind == domain.KindOpenApiConnector || c.Kind == domain.KindGraphqlConnector {
		return p.registerSpecConnector(ctx, c)
	}

	headers, err := p.renderer.Render(ctx, c.Meta.Namespace, c.Spec.Headers)
	if err != nil {
		return nil, fmt.Errorf("proxy: render credential for %q: %w", c.Meta.Name, err)
	}
	ep, err := p.runtime.Ensure(ctx, c, headers)
	if err != nil {
		return nil, fmt.Errorf("proxy: ensure runtime for %q: %w", c.Meta.Name, err)
	}
	cp, err := dial(ctx, c, ep, headers, p.authz, p)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if old := p.conns[c.Meta.Name]; old != nil {
		old.close()
	}
	p.conns[c.Meta.Name] = cp
	p.rebuildAggregate()
	p.mu.Unlock()

	p.log.Info("connector registered", "connector", c.Meta.Name, "url", "/mcp/"+c.Meta.Name, "tools", len(cp.tools))
	return cp.tools, nil
}

// registerSpecConnector compiles an OpenApiConnector or GraphqlConnector and
// registers it in-process via RegisterServer.
func (p *Proxy) registerSpecConnector(ctx context.Context, c *domain.Connector) ([]domain.Tool, error) {
	var tools []domain.Tool
	var handlers map[string]mcp.ToolHandler
	
	switch c.Kind {
	case domain.KindOpenApiConnector:
		conn, err := specconn.NewOpenAPIConnector(ctx, c.Meta.Name, c.Spec.SpecURL, c.Spec.BaseURL, c.Spec.Headers, p.renderer)
		if err != nil {
			return nil, fmt.Errorf("proxy: create OpenAPI connector %q: %w", c.Meta.Name, err)
		}
		tools, handlers, err = conn.CompileTools(c.Spec.Tools)
		if err != nil {
			return nil, fmt.Errorf("proxy: compile OpenAPI tools for %q: %w", c.Meta.Name, err)
		}
		
	case domain.KindGraphqlConnector:
		conn, err := specconn.NewGraphQLConnector(ctx, c.Meta.Name, c.Spec.Endpoint, c.Spec.Headers, p.renderer)
		if err != nil {
			return nil, fmt.Errorf("proxy: create GraphQL connector %q: %w", c.Meta.Name, err)
		}
		tools, handlers, err = conn.CompileTools(c.Spec.Tools)
		if err != nil {
			return nil, fmt.Errorf("proxy: compile GraphQL tools for %q: %w", c.Meta.Name, err)
		}
		
	default:
		return nil, fmt.Errorf("proxy: unknown spec connector kind %q", c.Kind)
	}
	
	// Register the compiled tools in-process.
	if err := p.RegisterServer(ctx, c.Meta.Name, handlers, tools); err != nil {
		return nil, fmt.Errorf("proxy: register spec connector %q: %w", c.Meta.Name, err)
	}
	
	return tools, nil
}

// RegisterServer registers an in-process connector with pre-built tool handlers.
// Unlike Register, it does not dial an upstream; the caller provides tool
// handlers that run in-process. This path is used by spec connectors (compiled
// tools) that run in-process without an upstream dial.
func (p *Proxy) RegisterServer(ctx context.Context, name string, handlers map[string]mcp.ToolHandler, tools []domain.Tool) error {
	srv := mcp.NewServer(&mcp.Implementation{Name: "tennon/" + name, Version: "0.1.0"}, nil)
	for _, t := range tools {
		h := handlers[t.Name]
		if h == nil {
			return fmt.Errorf("proxy: no handler for tool %q", t.Name)
		}
		srv.AddTool(&mcp.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: json.RawMessage(t.InputSchema),
		}, h)
	}

	cp := &connProxy{
		name:       name,
		server:     srv,
		tools:      tools,
		handler:    mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil),
		authz:      p.authz,
		inProcess:  handlers,
		proxy:      p,
	}

	p.mu.Lock()
	if old := p.conns[name]; old != nil {
		old.close()
	}
	p.conns[name] = cp
	p.rebuildAggregate()
	p.mu.Unlock()

	p.log.Info("in-process connector registered", "connector", name, "url", "/mcp/"+name, "tools", len(tools))
	return nil
}

// Deregister removes a connector's route and tears down its upstream.
func (p *Proxy) Deregister(ctx context.Context, c *domain.Connector) {
	p.mu.Lock()
	cp := p.conns[c.Meta.Name]
	delete(p.conns, c.Meta.Name)
	p.rebuildAggregate()
	p.mu.Unlock()
	if cp != nil {
		cp.close()
	}
	_ = p.runtime.Teardown(ctx, c)
	p.log.Info("connector deregistered", "connector", c.Meta.Name)
}

// Tools returns the currently exposed tools across all connectors.
func (p *Proxy) Tools() []domain.Tool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []domain.Tool
	for _, cp := range p.conns {
		out = append(out, cp.tools...)
	}
	return out
}

// rebuildAggregate rebuilds the cached aggregate server. Must be called with
// p.mu held.
func (p *Proxy) rebuildAggregate() {
	srv := mcp.NewServer(&mcp.Implementation{Name: "tennon/aggregate", Version: "0.1.0"}, nil)
	for _, cp := range p.conns {
		for _, t := range cp.tools {
			var handler mcp.ToolHandler
			if cp.inProcess != nil {
				// In-process connector: reuse the handler.
				handler = cp.inProcess[t.Name]
			} else {
				// Upstreamed connector: use the forward wrapper.
				handler = cp.forward(t.Upstream, t.Name)
			}
			srv.AddTool(&mcp.Tool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: json.RawMessage(t.InputSchema),
			}, handler)
		}
	}
	p.aggregate = &aggregateServer{
		server:  srv,
		handler: mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil),
	}
}

// Handler returns the proxy's HTTP handler, wrapped with auth middleware. Mount
// it at the proxy address.
func (p *Proxy) Handler() http.Handler {
	return p.authMiddleware(http.HandlerFunc(p.serve))
}

// serve dispatches requests: /mcp (global aggregate) or /mcp/<name> (per-connector).
func (p *Proxy) serve(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/mcp/")
	name = strings.SplitN(name, "/", 2)[0]
	if name == "" {
		// Global aggregate endpoint.
		p.serveAggregate(w, r)
		return
	}
	p.mu.RLock()
	cp := p.conns[name]
	p.mu.RUnlock()
	if cp == nil {
		http.Error(w, fmt.Sprintf("tennon proxy: no connector %q", name), http.StatusNotFound)
		return
	}
	cp.handler.ServeHTTP(w, r)
}

// serveAggregate serves /mcp with a merged MCP server exposing all connectors' tools.
func (p *Proxy) serveAggregate(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	agg := p.aggregate
	p.mu.RUnlock()

	if agg == nil {
		http.Error(w, "tennon proxy: no connectors registered", http.StatusNotFound)
		return
	}
	agg.handler.ServeHTTP(w, r)
}

// authMiddleware resolves the bearer credential to a subject and stores it in
// the request context for the tool handlers.
func (p *Proxy) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		sub, err := p.verifier.Verify(r.Context(), bearer)
		if err != nil {
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), subjectKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// createPendingExecution creates a pending Execution in the store and returns
// its id. The execution records the tool call that needs approval. The pending
// map holds a channel that will receive the result when the execution is approved.
//
// Design: The proxy holds pending calls in memory (execution id → result channel).
// On approval, the API calls CompleteExecution, which resumes the call to the
// upstream and sends the result on the channel. If the process restarts before
// approval, the Execution record persists but the in-memory state is lost — the
// caller will need to retry the call after approval. This is acceptable for Phase 5;
// a production system would use a durable queue.
func (p *Proxy) createPendingExecution(ctx context.Context, sub domain.Subject, connector, tool, upstreamTool string, args map[string]any, rawArgs json.RawMessage) string {
	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())
	exec := &domain.Execution{
		Meta: domain.ObjectMeta{
			Name:      execID,
			Namespace: "tennon",
		},
		Spec: domain.ExecutionSpec{
			Subject:   fmt.Sprintf("%s:%s", sub.Kind, sub.ID),
			Connector: connector,
			Tool:      tool,
			Arguments: args,
		},
		Status: domain.ExecutionStatus{
			State:   domain.ExecutionPending,
			Created: time.Now(),
		},
	}
	res, err := exec.ToResource()
	if err != nil {
		p.log.Error("failed to marshal execution", "err", err)
		return execID
	}
	if _, err := p.store.Apply(ctx, res); err != nil {
		p.log.Error("failed to persist execution", "err", err)
	}
	p.log.Info("execution paused", "id", execID, "connector", connector, "tool", tool)
	return execID
}

// ApproveExecution marks an execution as approved and completes the upstream call.
// It looks up the connector and calls the upstream tool, then updates the execution
// status with the result.
func (p *Proxy) ApproveExecution(ctx context.Context, execID string) error {
	res, err := p.store.Get(ctx, domain.KindExecution, "tennon", execID)
	if err != nil {
		return fmt.Errorf("get execution: %w", err)
	}
	exec, err := domain.ExecutionFromResource(res)
	if err != nil {
		return fmt.Errorf("unmarshal execution: %w", err)
	}
	if exec.Status.State != domain.ExecutionPending {
		return fmt.Errorf("execution %s is not pending (state: %s)", execID, exec.Status.State)
	}

	// Update status to approved.
	exec.Status.State = domain.ExecutionApproved
	exec.Status.Approved = time.Now()
	if err := p.updateExecutionStatus(ctx, exec); err != nil {
		return fmt.Errorf("update execution status: %w", err)
	}

	// Execute the upstream call.
	p.mu.RLock()
	cp := p.conns[exec.Spec.Connector]
	p.mu.RUnlock()
	if cp == nil {
		exec.Status.State = domain.ExecutionFailed
		exec.Status.Error = fmt.Sprintf("connector %s not found", exec.Spec.Connector)
		_ = p.updateExecutionStatus(ctx, exec)
		return fmt.Errorf("connector %s not found", exec.Spec.Connector)
	}

	// Find the upstream tool name from the exposed tool name.
	var upstreamName string
	for _, t := range cp.tools {
		if t.Name == exec.Spec.Tool {
			upstreamName = t.Upstream
			break
		}
	}
	if upstreamName == "" {
		exec.Status.State = domain.ExecutionFailed
		exec.Status.Error = fmt.Sprintf("tool %s not found on connector %s", exec.Spec.Tool, exec.Spec.Connector)
		_ = p.updateExecutionStatus(ctx, exec)
		return fmt.Errorf("tool %s not found", exec.Spec.Tool)
	}

	rawArgs, _ := json.Marshal(exec.Spec.Arguments)
	result, err := cp.upstream.CallTool(ctx, &mcp.CallToolParams{
		Name:      upstreamName,
		Arguments: rawArgs,
	})

	exec.Status.Completed = time.Now()
	if err != nil {
		exec.Status.State = domain.ExecutionFailed
		exec.Status.Error = err.Error()
	} else {
		exec.Status.State = domain.ExecutionCompleted
		resultJSON, _ := json.Marshal(result)
		exec.Status.Result = resultJSON
	}
	_ = p.updateExecutionStatus(ctx, exec)

	p.log.Info("execution completed", "id", execID, "state", exec.Status.State)
	return nil
}

// DenyExecution marks an execution as denied.
func (p *Proxy) DenyExecution(ctx context.Context, execID string) error {
	res, err := p.store.Get(ctx, domain.KindExecution, "tennon", execID)
	if err != nil {
		return fmt.Errorf("get execution: %w", err)
	}
	exec, err := domain.ExecutionFromResource(res)
	if err != nil {
		return fmt.Errorf("unmarshal execution: %w", err)
	}
	if exec.Status.State != domain.ExecutionPending {
		return fmt.Errorf("execution %s is not pending (state: %s)", execID, exec.Status.State)
	}

	exec.Status.State = domain.ExecutionDenied
	exec.Status.Denied = time.Now()
	if err := p.updateExecutionStatus(ctx, exec); err != nil {
		return fmt.Errorf("update execution status: %w", err)
	}

	p.log.Info("execution denied", "id", execID)
	return nil
}

// ListExecutions returns all executions, optionally filtered by state.
func (p *Proxy) ListExecutions(ctx context.Context, state string) ([]*domain.Execution, error) {
	resources, err := p.store.List(ctx, domain.KindExecution, "")
	if err != nil {
		return nil, err
	}
	var out []*domain.Execution
	for _, res := range resources {
		exec, err := domain.ExecutionFromResource(res)
		if err != nil {
			return nil, err
		}
		if state == "" || string(exec.Status.State) == state {
			out = append(out, exec)
		}
	}
	return out, nil
}

// GetExecution returns a single execution by id.
func (p *Proxy) GetExecution(ctx context.Context, execID string) (*domain.Execution, error) {
	res, err := p.store.Get(ctx, domain.KindExecution, "tennon", execID)
	if err != nil {
		return nil, err
	}
	return domain.ExecutionFromResource(res)
}

func (p *Proxy) updateExecutionStatus(ctx context.Context, exec *domain.Execution) error {
	res, err := exec.ToResource()
	if err != nil {
		return err
	}
	return p.store.UpdateStatus(ctx, res)
}
