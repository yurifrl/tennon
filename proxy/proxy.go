// Package proxy is Tennon's data plane and MCP surface. Every connector is
// reachable at /mcp/<name> over streamable-HTTP. On each request the proxy
// authenticates the caller, authorizes the call, renders the connector's
// credential in-process, and forwards to the upstream MCP server. The caller
// never sees the credential.
package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/yurifrl/tennon/domain"
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
	log      *slog.Logger

	mu    sync.RWMutex
	conns map[string]*connProxy
}

// New builds a Proxy. The concrete ports are supplied by the composition root.
func New(renderer domain.SecretRenderer, rt domain.Runtime, authz domain.Authorizer, verifier domain.IdentityVerifier, log *slog.Logger) *Proxy {
	if log == nil {
		log = slog.Default()
	}
	return &Proxy{
		renderer: renderer,
		runtime:  rt,
		authz:    authz,
		verifier: verifier,
		log:      log,
		conns:    map[string]*connProxy{},
	}
}

// Register makes a connector reachable: it renders the credential, ensures the
// upstream runtime, dials it, discovers tools, and installs the route. It
// returns the discovered tools so the controller can update status.
func (p *Proxy) Register(ctx context.Context, c *domain.Connector) ([]domain.Tool, error) {
	headers, err := p.renderer.Render(ctx, c.Meta.Namespace, c.Spec.Headers)
	if err != nil {
		return nil, fmt.Errorf("proxy: render credential for %q: %w", c.Meta.Name, err)
	}
	ep, err := p.runtime.Ensure(ctx, c, headers)
	if err != nil {
		return nil, fmt.Errorf("proxy: ensure runtime for %q: %w", c.Meta.Name, err)
	}
	cp, err := dial(ctx, c, ep, headers, p.authz)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if old := p.conns[c.Meta.Name]; old != nil {
		old.close()
	}
	p.conns[c.Meta.Name] = cp
	p.mu.Unlock()

	p.log.Info("connector registered", "connector", c.Meta.Name, "url", "/mcp/"+c.Meta.Name, "tools", len(cp.tools))
	return cp.tools, nil
}

// Deregister removes a connector's route and tears down its upstream.
func (p *Proxy) Deregister(ctx context.Context, c *domain.Connector) {
	p.mu.Lock()
	cp := p.conns[c.Meta.Name]
	delete(p.conns, c.Meta.Name)
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

// Handler returns the proxy's HTTP handler, wrapped with auth middleware. Mount
// it at the proxy address.
func (p *Proxy) Handler() http.Handler {
	return p.authMiddleware(http.HandlerFunc(p.serve))
}

// serve dispatches /mcp/<name> to the matching connector proxy.
func (p *Proxy) serve(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/mcp/")
	name = strings.SplitN(name, "/", 2)[0]
	if name == "" {
		http.Error(w, "tennon proxy: specify a connector at /mcp/<name>", http.StatusNotFound)
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
