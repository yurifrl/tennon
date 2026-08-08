package specconn_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/secret"
	"github.com/yurifrl/tennon/specconn"
	"github.com/yurifrl/tennon/store"
)

// TestOpenAPIConnectorWithMockBackend verifies that:
// 1. An OpenAPI spec is compiled into tools
// 2. Tool calls inject rendered credentials
// 3. Credentials are not exposed in tool definitions or responses
func TestOpenAPIConnectorWithMockBackend(t *testing.T) {
	ctx := context.Background()
	
	// Setup secret store and renderer.
	s := store.NewMemory()
	secretRes, err := domain.NewSecretResource("default", "api-creds", map[string]string{
		"token": "secret-api-token-12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, secretRes); err != nil {
		t.Fatal(err)
	}
	renderer := secret.NewRenderer(s, "default")
	
	// Create mock backend that records received headers.
	var receivedAuth string
	var receivedBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			receivedBody = string(bodyBytes)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer backend.Close()
	
	// Create minimal OpenAPI spec.
	specContent := `{
		"openapi": "3.0.0",
		"info": {"title": "Test API", "version": "1.0.0"},
		"servers": [{"url": "` + backend.URL + `"}],
		"paths": {
			"/users/{userId}": {
				"get": {
					"operationId": "getUser",
					"summary": "Get user by ID",
					"parameters": [{
						"name": "userId",
						"in": "path",
						"required": true,
						"schema": {"type": "string"}
					}],
					"responses": {
						"200": {
							"description": "Success"
						}
					}
				}
			},
			"/posts": {
				"post": {
					"operationId": "createPost",
					"summary": "Create a post",
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"title": {"type": "string"},
										"content": {"type": "string"}
									}
								}
							}
						}
					},
					"responses": {
						"201": {
							"description": "Created"
						}
					}
				}
			}
		}
	}`
	
	specServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(specContent))
	}))
	defer specServer.Close()
	
	// Create OpenAPI connector with credential template.
	headers := map[string]string{
		"Authorization": "Bearer {{secret:api-creds.token}}",
	}
	connector, err := specconn.NewOpenAPIConnector(ctx, "test-api", specServer.URL, backend.URL, headers, renderer)
	if err != nil {
		t.Fatalf("NewOpenAPIConnector failed: %v", err)
	}
	
	// Compile tools - allow all (empty ToolsSpec permits all).
	toolsSpec := domain.ToolsSpec{}
	tools, handlers, err := connector.CompileTools(toolsSpec)
	if err != nil {
		t.Fatalf("CompileTools failed: %v", err)
	}
	
	// Verify tools were generated.
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	
	// Verify tool definitions don't expose the secret.
	for _, tool := range tools {
		toolJSON, _ := json.Marshal(tool)
		toolStr := string(toolJSON)
		if containsSecret(toolStr) {
			t.Errorf("tool definition exposes secret: %s", toolStr)
		}
	}
	
	// Find the getUser handler.
	getUserHandler, ok := handlers["getUser"]
	if !ok {
		t.Fatal("getUser handler not found")
	}
	
	// Call the tool.
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "getUser",
			Arguments: json.RawMessage(`{"userId": "123"}`),
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
	expectedAuth := "Bearer secret-api-token-12345"
	if receivedAuth != expectedAuth {
		t.Errorf("backend received Authorization = %q, want %q", receivedAuth, expectedAuth)
	}
	
	// Verify credential is NOT exposed in the response.
	resultJSON, _ := json.Marshal(result)
	if containsSecret(string(resultJSON)) {
		t.Errorf("response exposes secret: %s", string(resultJSON))
	}
	
	// Test POST with body.
	createPostHandler, ok := handlers["createPost"]
	if !ok {
		t.Fatal("createPost handler not found")
	}
	
	receivedAuth = ""
	receivedBody = ""
	
	postReq := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name: "createPost",
			Arguments: json.RawMessage(`{"body": {"title": "Test", "content": "Hello"}}`),
		},
	}
	
	result, err = createPostHandler(ctx, postReq)
	if err != nil {
		t.Fatalf("createPost handler failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("createPost returned error: %+v", result)
	}
	
	// Verify credential was injected for POST too.
	if receivedAuth != expectedAuth {
		t.Errorf("POST backend received Authorization = %q, want %q", receivedAuth, expectedAuth)
	}
	
	// Verify body was sent.
	if receivedBody == "" {
		t.Error("POST body was not sent")
	}
}

func containsSecret(s string) bool {
	// Check if the actual secret value appears in the string.
	return contains(s, "secret-api-token-12345")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
