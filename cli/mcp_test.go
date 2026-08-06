package cli_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/cli"
	"github.com/yurifrl/tennon/domain"
)

// TestMCPServer verifies the embedded MCP server exposes CLI operations as tools
// and successfully bridges to the Tennon API.
func TestMCPServer(t *testing.T) {
	// Mock Tennon API that returns a connector list
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/connectors" && r.Method == "GET" {
			resp := struct {
				Items []domain.Connector `json:"items"`
			}{
				Items: []domain.Connector{
					{
						Meta: domain.ObjectMeta{Name: "test-connector"},
						Spec: domain.ConnectorSpec{Description: "test"},
						Status: domain.ConnectorStatus{
							Phase: "Ready",
							URL:   "http://localhost:9090",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer apiServer.Close()

	// Create MCP server pointing at mock API
	mcpServer := cli.NewMCPServer(apiServer.URL, "test-token")

	// Create in-memory transports for testing
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- mcpServer.Run(ctx, serverTransport)
	}()

	// Connect client
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	// List tools
	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	// Verify expected tools are present
	expectedTools := map[string]bool{
		"tennon_connector_list":   false,
		"tennon_connector_create": false,
		"tennon_secret_create":    false,
		"tennon_tool_call":        false,
		"tennon_policy_manage":    false,
	}
	for _, tool := range tools.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}
	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool %q not found", name)
		}
	}

	// Call tennon_connector_list tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "tennon_connector_list",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(tennon_connector_list) failed: %v", err)
	}

	// Verify result contains connector data
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content from tennon_connector_list")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if textContent.Text == "" {
		t.Fatal("expected non-empty text in result")
	}

	// Verify the result contains our test connector
	var connectors []domain.Connector
	if err := json.Unmarshal([]byte(textContent.Text), &connectors); err != nil {
		t.Fatalf("failed to parse connector list: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	if connectors[0].Meta.Name != "test-connector" {
		t.Errorf("expected connector name 'test-connector', got %q", connectors[0].Meta.Name)
	}
}
