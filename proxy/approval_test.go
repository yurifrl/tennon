package proxy_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// TestApprovalFlow tests the pause/resume human-in-the-loop flow:
// 1. A tool call that resolves to 'ask' pauses and returns an execution id
// 2. The execution is listable and in pending state
// 3. Approving the execution completes the upstream call
// 4. The execution status is updated to completed
func TestApprovalFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Mock upstream MCP server with a "sensitive_action" tool
	upstream := &mockUpstream{}
	upSrv := httptest.NewServer(upstream.handler())
	defer upSrv.Close()

	// 2. Store with a Policy that sets EffectAsk for delete_all
	st := store.NewMemory()
	policy := &domain.Policy{
		Meta: domain.ObjectMeta{Name: "approval-required", Namespace: "tennon"},
		Spec: domain.PolicySpec{
			Rules: []domain.Rule{
				{
					Resource: domain.ResourceMatch{Connector: "test", Tool: "get_status"},
					Effect:   domain.EffectAllow,
				},
				{
					Resource: domain.ResourceMatch{Connector: "test", Tool: "delete_all"},
					Effect:   domain.EffectAsk,
				},
			},
		},
	}
	policyRes, _ := policy.ToResource()
	if _, err := st.Apply(ctx, policyRes); err != nil {
		t.Fatal(err)
	}

	// 3. Create authorizer that uses the policy
	authz, err := auth.NewPolicyAuthorizer(ctx, st, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	// 4. Proxy wired with policy-based authorizer
	px := proxy.New(
		secret.NewRenderer(st, "tennon"),
		runtime.NewMulti(runtime.NewRemote()),
		authz,
		auth.NewAnonymousVerifier(),
		st,
		nil,
	)

	// 5. Register connector
	conn := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "test", Namespace: "tennon"},
		Spec: domain.ConnectorSpec{
			Runtime:   domain.RuntimeSpec{Remote: &domain.RemoteRuntime{URL: upSrv.URL}},
			Transport: "streamable-http",
		},
	}
	if _, err := px.Register(ctx, conn); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// 6. Serve proxy
	proxySrv := httptest.NewServer(px.Handler())
	defer proxySrv.Close()

	// 7. Connect client and try to call delete_all (should pause)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             proxySrv.URL + "/mcp/test",
		HTTPClient:           http.DefaultClient,
		DisableStandaloneSSE: true,
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "delete_all",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	// 8. Result should contain "approval required" and an execution id
	// Note: The result is not an error, just a paused execution
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	t.Logf("Result text: %s", tc.Text)
	if !containsStr(tc.Text, "approval required") || !containsStr(tc.Text, "exec-") {
		t.Fatalf("expected approval required message with execution id, got: %s", tc.Text)
	}

	// 9. List executions - should have one pending
	execs, err := px.ListExecutions(ctx, "pending")
	if err != nil {
		t.Fatalf("ListExecutions: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 pending execution, got %d", len(execs))
	}
	exec := execs[0]
	if exec.Status.State != domain.ExecutionPending {
		t.Fatalf("expected pending state, got %s", exec.Status.State)
	}
	if exec.Spec.Tool != "delete_all" {
		t.Fatalf("expected tool delete_all, got %s", exec.Spec.Tool)
	}

	// 10. Approve the execution
	if err := px.ApproveExecution(ctx, exec.Meta.Name); err != nil {
		t.Fatalf("ApproveExecution: %v", err)
	}

	// 11. Get the execution - should be completed
	completedExec, err := px.GetExecution(ctx, exec.Meta.Name)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if completedExec.Status.State != domain.ExecutionCompleted {
		t.Fatalf("expected completed state, got %s", completedExec.Status.State)
	}
	if len(completedExec.Status.Result) == 0 {
		t.Fatal("expected result in completed execution")
	}

	// 12. Verify the result is the upstream response
	var toolResult mcp.CallToolResult
	if err := json.Unmarshal(completedExec.Status.Result, &toolResult); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(toolResult.Content) == 0 {
		t.Fatal("expected content in tool result")
	}
	resultText, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", toolResult.Content[0])
	}
	if resultText.Text != "deleted" {
		t.Fatalf("expected 'deleted', got %s", resultText.Text)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
