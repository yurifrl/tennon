package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
)

// connProxy is the proxy state for a single connector: a live session to the
// upstream MCP server and an MCP server that re-exposes the upstream's tools
// (prefixed and filtered) at /mcp/<name>.
type connProxy struct {
	name     string
	spec     domain.ConnectorSpec
	upstream *mcp.ClientSession
	server   *mcp.Server
	handler  http.Handler
	tools    []domain.Tool
	authz    domain.Authorizer
	cancel   context.CancelFunc
}

// headerRoundTripper injects the rendered credential headers into every request
// to the upstream MCP server. The values never leave this process.
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// dial opens a session to the upstream and builds the re-exposing server. The
// upstream session gets its own long-lived context so it outlives the caller's
// Register request; it is cancelled by close.
func dial(ctx context.Context, c *domain.Connector, ep *domain.Endpoint, headers map[string]string, authz domain.Authorizer) (*connProxy, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "tennon-proxy", Version: "0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             ep.URL,
		HTTPClient:           &http.Client{Transport: headerRoundTripper{headers: headers}},
		DisableStandaloneSSE: true, // request/response only; no persistent server-push stream
	}
	sessionCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	session, err := client.Connect(sessionCtx, transport, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("proxy: connect upstream %q at %s: %w", c.Meta.Name, ep.URL, err)
	}

	cp := &connProxy{name: c.Meta.Name, spec: c.Spec, upstream: session, authz: authz, cancel: cancel}

	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		session.Close()
		cancel()
		return nil, fmt.Errorf("proxy: list tools for %q: %w", c.Meta.Name, err)
	}

	srv := mcp.NewServer(&mcp.Implementation{Name: "tennon/" + c.Meta.Name, Version: "0.1.0"}, nil)
	for _, ut := range listed.Tools {
		if !c.Spec.Tools.Permits(ut.Name) {
			continue
		}
		exposed := c.Spec.Tools.ExposedName(ut.Name)
		schema, _ := json.Marshal(ut.InputSchema)
		cp.tools = append(cp.tools, domain.Tool{
			Name:        exposed,
			Upstream:    ut.Name,
			Description: ut.Description,
			InputSchema: schema,
			Connector:   c.Meta.Name,
			Tags:        c.Spec.Tags,
		})
		srv.AddTool(&mcp.Tool{
			Name:        exposed,
			Description: ut.Description,
			InputSchema: ut.InputSchema,
		}, cp.forward(ut.Name, exposed))
	}

	cp.server = srv
	cp.handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	return cp, nil
}

// forward returns a handler that authorizes then relays the call to the
// upstream MCP server, translating the exposed name back to the upstream name.
func (cp *connProxy) forward(upstreamName, exposedName string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sub := SubjectFromContext(ctx)
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			_ = json.Unmarshal(req.Params.Arguments, &args)
		}
		decision := cp.authz.Decide(ctx, sub, domain.ToolCall{
			Connector: cp.name, Tool: exposedName, Arguments: args,
		})
		switch decision.Effect {
		case domain.EffectBlock:
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "blocked by policy: " + decision.Reason}},
			}, nil
		case domain.EffectAsk:
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "approval required (human-in-the-loop not yet available): " + decision.Reason}},
			}, nil
		}
		result, err := cp.upstream.CallTool(ctx, &mcp.CallToolParams{
			Name:      upstreamName,
			Arguments: json.RawMessage(req.Params.Arguments),
		})
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "upstream error: " + err.Error()}},
			}, nil
		}
		return result, nil
	}
}

func (cp *connProxy) close() {
	if cp.upstream != nil {
		cp.upstream.Close()
	}
	if cp.cancel != nil {
		cp.cancel()
	}
}
