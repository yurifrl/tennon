package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yurifrl/tennon/auth"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/proxy"
	"github.com/yurifrl/tennon/runtime"
	"github.com/yurifrl/tennon/secret"
	"github.com/yurifrl/tennon/store"
)

// mockUpstream is an MCP server exposing get_status and delete_all. It records
// the Authorization header it received so the test can assert the credential
// was injected by the proxy.
type mockUpstream struct {
	mu       sync.Mutex
	lastAuth string
}

func (m *mockUpstream) handler() http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "mock", Version: "1.0.0"}, nil)
	reply := func(text string) mcp.ToolHandler {
		return func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
		}
	}
	srv.AddTool(&mcp.Tool{Name: "get_status", Description: "read status", InputSchema: emptySchema()}, reply("status: ok"))
	srv.AddTool(&mcp.Tool{Name: "delete_all", Description: "danger", InputSchema: emptySchema()}, reply("deleted"))

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.lastAuth = r.Header.Get("Authorization")
		m.mu.Unlock()
		mcpHandler.ServeHTTP(w, r)
	})
}

func emptySchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func TestProxyEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Mock upstream MCP server.
	up := &mockUpstream{}
	upSrv := httptest.NewServer(up.handler())
	defer upSrv.Close()

	// 2. Store with the connector's credential secret.
	st := store.NewMemory()
	secRes, _ := domain.NewSecretResource("tennon", "grafana-creds", map[string]string{"token": "topsecret"})
	if _, err := st.Apply(ctx, secRes); err != nil {
		t.Fatal(err)
	}

	// 3. Proxy wired with real renderer + remote runtime + phase-1 defaults.
	px := proxy.New(
		secret.NewRenderer(st, "tennon"),
		runtime.NewMulti(runtime.NewRemote()),
		auth.NewAllowAllAuthorizer(),
		auth.NewAnonymousVerifier(),
		nil,
	)

	// 4. Register a connector: remote upstream, credential header, prefix, allow only get_*.
	conn := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "grafana", Namespace: "tennon"},
		Spec: domain.ConnectorSpec{
			Runtime:   domain.RuntimeSpec{Remote: &domain.RemoteRuntime{URL: upSrv.URL}},
			Transport: "streamable-http",
			Headers:   map[string]string{"Authorization": "Bearer {{secret:grafana-creds.token}}"},
			Tools:     domain.ToolsSpec{ExposePrefix: "grafana_", Allow: []string{"get_*"}},
		},
	}
	tools, err := px.Register(ctx, conn)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// allow filter drops delete_all; only get_status remains, prefixed.
	if len(tools) != 1 || tools[0].Name != "grafana_get_status" {
		t.Fatalf("exposed tools = %+v, want [grafana_get_status]", tools)
	}

	// 5. Serve the proxy and connect a real MCP client to /mcp/grafana.
	proxySrv := httptest.NewServer(px.Handler())
	defer proxySrv.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: proxySrv.URL + "/mcp/grafana", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// 6. The caller sees only the prefixed, allowed tool.
	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(listed.Tools) != 1 || listed.Tools[0].Name != "grafana_get_status" {
		t.Fatalf("client sees tools = %v", toolNames(listed.Tools))
	}

	// 7. Call the tool through the proxy.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "grafana_get_status"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("tool error: %+v", res)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "status: ok" {
		t.Fatalf("unexpected result content: %+v", res.Content[0])
	}

	// 8. The credential was rendered and injected into the upstream call...
	up.mu.Lock()
	gotAuth := up.lastAuth
	up.mu.Unlock()
	if gotAuth != "Bearer topsecret" {
		t.Fatalf("upstream Authorization = %q, want %q", gotAuth, "Bearer topsecret")
	}
	// ...and the caller never presented it (the agent used no Authorization header).
}

func toolNames(tools []*mcp.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Name
	}
	return out
}

func TestProxyAggregateEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Two mock upstream MCP servers.
	up1 := &mockUpstream{}
	upSrv1 := httptest.NewServer(up1.handler())
	defer upSrv1.Close()

	up2 := &mockUpstream{}
	upSrv2 := httptest.NewServer(up2.handler())
	defer upSrv2.Close()

	// 2. Store with credentials.
	st := store.NewMemory()
	sec1, _ := domain.NewSecretResource("tennon", "creds-1", map[string]string{"token": "secret1"})
	sec2, _ := domain.NewSecretResource("tennon", "creds-2", map[string]string{"token": "secret2"})
	if _, err := st.Apply(ctx, sec1); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Apply(ctx, sec2); err != nil {
		t.Fatal(err)
	}

	// 3. Proxy.
	px := proxy.New(
		secret.NewRenderer(st, "tennon"),
		runtime.NewMulti(runtime.NewRemote()),
		auth.NewAllowAllAuthorizer(),
		auth.NewAnonymousVerifier(),
		nil,
	)

	// 4. Register two connectors with different prefixes.
	conn1 := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "grafana", Namespace: "tennon"},
		Spec: domain.ConnectorSpec{
			Runtime:   domain.RuntimeSpec{Remote: &domain.RemoteRuntime{URL: upSrv1.URL}},
			Transport: "streamable-http",
			Headers:   map[string]string{"Authorization": "Bearer {{secret:creds-1.token}}"},
			Tools:     domain.ToolsSpec{ExposePrefix: "grafana_", Allow: []string{"get_*"}},
		},
	}
	tools1, err := px.Register(ctx, conn1)
	if err != nil {
		t.Fatalf("Register conn1: %v", err)
	}
	if len(tools1) != 1 || tools1[0].Name != "grafana_get_status" {
		t.Fatalf("conn1 tools = %+v", tools1)
	}

	conn2 := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "github", Namespace: "tennon"},
		Spec: domain.ConnectorSpec{
			Runtime:   domain.RuntimeSpec{Remote: &domain.RemoteRuntime{URL: upSrv2.URL}},
			Transport: "streamable-http",
			Headers:   map[string]string{"Authorization": "Bearer {{secret:creds-2.token}}"},
			Tools:     domain.ToolsSpec{ExposePrefix: "github_", Allow: []string{"*"}},
		},
	}
	tools2, err := px.Register(ctx, conn2)
	if err != nil {
		t.Fatalf("Register conn2: %v", err)
	}
	if len(tools2) != 2 {
		t.Fatalf("conn2 tools = %+v, want 2 tools", tools2)
	}

	// 5. Serve the proxy and connect to the global /mcp endpoint.
	proxySrv := httptest.NewServer(px.Handler())
	defer proxySrv.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: proxySrv.URL + "/mcp", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client connect to /mcp: %v", err)
	}
	defer session.Close()

	// 6. The aggregate endpoint lists tools from both connectors.
	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := toolNames(listed.Tools)
	if len(names) != 3 {
		t.Fatalf("aggregate tools = %v, want 3 tools (grafana_get_status + 2 github tools)", names)
	}
	// Check that grafana_get_status is present.
	hasGrafana := false
	for _, name := range names {
		if name == "grafana_get_status" {
			hasGrafana = true
		}
	}
	if !hasGrafana {
		t.Fatalf("aggregate tools = %v, missing grafana_get_status", names)
	}

	// 7. Call a tool through the aggregate endpoint - it routes to the correct connector.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "grafana_get_status"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("tool error: %+v", res)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "status: ok" {
		t.Fatalf("unexpected result content: %+v", res.Content[0])
	}
}

func TestProxyInProcessServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Build tool handlers for an in-process connector.
	reply := func(text string) mcp.ToolHandler {
		return func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
		}
	}
	handlers := map[string]mcp.ToolHandler{
		"hello": reply("hello, world"),
	}

	// 2. Proxy.
	st := store.NewMemory()
	px := proxy.New(
		secret.NewRenderer(st, "tennon"),
		runtime.NewMulti(runtime.NewRemote()),
		auth.NewAllowAllAuthorizer(),
		auth.NewAnonymousVerifier(),
		nil,
	)

	// 3. Register the in-process connector.
	tools := []domain.Tool{{Name: "hello", Description: "greet", Connector: "local", InputSchema: emptySchema()}}
	if err := px.RegisterServer(ctx, "local", handlers, tools); err != nil {
		t.Fatalf("RegisterServer: %v", err)
	}

	// 4. Serve the proxy and connect to /mcp/local.
	proxySrv := httptest.NewServer(px.Handler())
	defer proxySrv.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: proxySrv.URL + "/mcp/local", DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client connect to /mcp/local: %v", err)
	}
	defer session.Close()

	// 5. List tools - should see the in-process tool.
	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(listed.Tools) != 1 || listed.Tools[0].Name != "hello" {
		t.Fatalf("tools = %v, want [hello]", toolNames(listed.Tools))
	}

	// 6. Call the tool.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "hello"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("tool error: %+v", res)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "hello, world" {
		t.Fatalf("unexpected result content: %+v", res.Content[0])
	}
}
