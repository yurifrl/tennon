package specconn_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/secret"
	"github.com/yurifrl/tennon/specconn"
	"github.com/yurifrl/tennon/store"
)

// TestGraphQLConnectorWithMockBackend verifies that:
// 1. GraphQL introspection compiles queries/mutations into tools
// 2. Tool calls inject rendered credentials
// 3. Credentials are not exposed in tool definitions or responses
func TestGraphQLConnectorWithMockBackend(t *testing.T) {
	ctx := context.Background()
	
	// Setup secret store and renderer.
	s := store.NewMemory()
	secretRes, err := domain.NewSecretResource("default", "graphql-creds", map[string]string{
		"apiKey": "secret-graphql-key-67890",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, secretRes); err != nil {
		t.Fatal(err)
	}
	renderer := secret.NewRenderer(s, "default")
	
	// Create mock GraphQL backend.
	var receivedAuth string
	var lastQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("X-API-Key")
		
		bodyBytes, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		json.Unmarshal(bodyBytes, &req)
		lastQuery = req.Query
		
		w.Header().Set("Content-Type", "application/json")
		
		// Handle introspection query.
		if strings.Contains(req.Query, "__schema") {
			introspectionResponse := map[string]any{
				"data": map[string]any{
					"__schema": map[string]any{
						"queryType": map[string]any{
							"name": "Query",
							"fields": []map[string]any{
								{
									"name":        "getUser",
									"description": "Fetch user by ID",
									"args": []map[string]any{
										{"name": "id", "type": map[string]any{"name": "String"}},
									},
								},
							},
						},
						"mutationType": map[string]any{
							"name": "Mutation",
							"fields": []map[string]any{
								{
									"name":        "createUser",
									"description": "Create a new user",
									"args": []map[string]any{
										{"name": "name", "type": map[string]any{"name": "String"}},
									},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(introspectionResponse)
			return
		}
		
		// Handle actual queries/mutations.
		response := map[string]any{
			"data": map[string]any{
				"result": "success",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer backend.Close()
	
	// Create GraphQL connector with credential template.
	headers := map[string]string{
		"X-API-Key": "{{secret:graphql-creds.apiKey}}",
	}
	connector, err := specconn.NewGraphQLConnector(ctx, "test-graphql", backend.URL, headers, renderer)
	if err != nil {
		t.Fatalf("NewGraphQLConnector failed: %v", err)
	}
	
	// Compile tools - allow all (empty ToolsSpec permits all).
	toolsSpec := domain.ToolsSpec{}
	tools, handlers, err := connector.CompileTools(toolsSpec)
	if err != nil {
		t.Fatalf("CompileTools failed: %v", err)
	}
	
	// Verify tools were generated (getUser query + createUser mutation).
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	
	// Verify tool definitions don't expose the secret.
	for _, tool := range tools {
		toolJSON, _ := json.Marshal(tool)
		toolStr := string(toolJSON)
		if containsGraphQLSecret(toolStr) {
			t.Errorf("tool definition exposes secret: %s", toolStr)
		}
	}
	
	// Find the getUser handler.
	getUserHandler, ok := handlers["getUser"]
	if !ok {
		t.Fatal("getUser handler not found")
	}
	
	// Call the query tool.
	receivedAuth = ""
	lastQuery = ""
	
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "getUser",
			Arguments: json.RawMessage(`{"id": "user-123"}`),
		},
	}
	
	result, err := getUserHandler(ctx, req)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error: %+v", result)
	}
	
	// Verify credential was injected into backend call.
	expectedAuth := "secret-graphql-key-67890"
	if receivedAuth != expectedAuth {
		t.Errorf("backend received X-API-Key = %q, want %q", receivedAuth, expectedAuth)
	}
	
	// Verify a GraphQL query was sent.
	if !strings.Contains(lastQuery, "getUser") {
		t.Errorf("query doesn't contain getUser: %s", lastQuery)
	}
	
	// Verify credential is NOT exposed in the response.
	resultJSON, _ := json.Marshal(result)
	if containsGraphQLSecret(string(resultJSON)) {
		t.Errorf("response exposes secret: %s", string(resultJSON))
	}
	
	// Test mutation.
	createUserHandler, ok := handlers["createUser"]
	if !ok {
		t.Fatal("createUser handler not found")
	}
	
	receivedAuth = ""
	lastQuery = ""
	
	mutationReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "createUser",
			Arguments: json.RawMessage(`{"name": "Alice"}`),
		},
	}
	
	result, err = createUserHandler(ctx, mutationReq)
	if err != nil {
		t.Fatalf("createUser handler failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("createUser returned error: %+v", result)
	}
	
	// Verify credential was injected for mutation too.
	if receivedAuth != expectedAuth {
		t.Errorf("mutation backend received X-API-Key = %q, want %q", receivedAuth, expectedAuth)
	}
	
	// Verify a GraphQL mutation was sent.
	if !strings.Contains(lastQuery, "createUser") && !strings.Contains(lastQuery, "mutation") {
		t.Errorf("query doesn't contain createUser or mutation: %s", lastQuery)
	}
}

func containsGraphQLSecret(s string) bool {
	return strings.Contains(s, "secret-graphql-key-67890")
}
