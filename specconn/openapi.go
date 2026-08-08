// Package specconn compiles OpenAPI and GraphQL specs into in-process MCP tool
// handlers. No upstream dial, no Pod — a tool call is just an HTTP request that
// Tennon makes on the caller's behalf with the rendered credential injected.
package specconn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
)

// OpenAPIConnector compiles an OpenAPI spec into tool handlers.
type OpenAPIConnector struct {
	name     string
	spec     *openapi3.T
	baseURL  string
	client   *http.Client
	renderer domain.SecretRenderer
	headers  map[string]string // credential headers with {{secret:...}} templates
}

// NewOpenAPIConnector fetches and parses an OpenAPI spec, returning a connector
// ready to compile tools.
func NewOpenAPIConnector(ctx context.Context, name string, specURL string, baseURL string, headers map[string]string, renderer domain.SecretRenderer) (*OpenAPIConnector, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	
	doc, err := loader.LoadFromURI(mustParseURL(specURL))
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec from %q: %w", specURL, err)
	}
	
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}
	
	// Use provided baseURL or fall back to first server in spec.
	if baseURL == "" && len(doc.Servers) > 0 {
		baseURL = doc.Servers[0].URL
	}
	if baseURL == "" {
		return nil, fmt.Errorf("no baseURL provided and no servers in spec")
	}
	
	return &OpenAPIConnector{
		name:     name,
		spec:     doc,
		baseURL:  baseURL,
		client:   &http.Client{},
		renderer: renderer,
		headers:  headers,
	}, nil
}

// CompileTools converts spec operations into domain.Tool definitions.
func (c *OpenAPIConnector) CompileTools(toolsSpec domain.ToolsSpec) ([]domain.Tool, map[string]mcp.ToolHandler, error) {
	var tools []domain.Tool
	handlers := make(map[string]mcp.ToolHandler)
	
	for path, pathItem := range c.spec.Paths.Map() {
		for method, op := range pathItem.Operations() {
			if op == nil || op.OperationID == "" {
				continue
			}
			
			toolName := op.OperationID
			if !toolsSpec.Permits(toolName) {
				continue
			}
			exposedName := toolsSpec.ExposedName(toolName)
			
			inputSchema, err := c.buildInputSchema(op)
			if err != nil {
				return nil, nil, fmt.Errorf("build schema for %s: %w", op.OperationID, err)
			}
			
			tools = append(tools, domain.Tool{
				Name:        exposedName,
				Upstream:    toolName,
				Description: op.Summary,
				InputSchema: inputSchema,
				Connector:   c.name,
			})
			
			handlers[exposedName] = c.makeHandler(method, path, op)
		}
	}
	
	return tools, handlers, nil
}

// buildInputSchema constructs a JSON schema from operation parameters and requestBody.
func (c *OpenAPIConnector) buildInputSchema(op *openapi3.Operation) (json.RawMessage, error) {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}
	
	required := []string{}
	props := schema["properties"].(map[string]any)
	
	// Add parameters as properties.
	for _, paramRef := range op.Parameters {
		if paramRef.Value == nil {
			continue
		}
		param := paramRef.Value
		if param.Schema != nil && param.Schema.Value != nil {
			props[param.Name] = param.Schema.Value
			if param.Required {
				required = append(required, param.Name)
			}
		}
	}
	
	// Add requestBody as a "body" property if present.
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, content := range op.RequestBody.Value.Content {
			if content.Schema != nil && content.Schema.Value != nil {
				props["body"] = content.Schema.Value
				if op.RequestBody.Value.Required {
					required = append(required, "body")
				}
				break
			}
		}
	}
	
	if len(required) > 0 {
		schema["required"] = required
	}
	
	return json.Marshal(schema)
}

// makeHandler returns a tool handler that executes the HTTP request with
// rendered credentials.
func (c *OpenAPIConnector) makeHandler(method string, path string, op *openapi3.Operation) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResult("invalid arguments: " + err.Error()), nil
			}
		}
		
		// Build URL with path parameters.
		url := c.baseURL + path
		for paramName, paramValue := range args {
			if paramName == "body" {
				continue
			}
			placeholder := "{" + paramName + "}"
			if strings.Contains(url, placeholder) {
				url = strings.ReplaceAll(url, placeholder, fmt.Sprint(paramValue))
			}
		}
		
		// Build request body.
		var body io.Reader
		if bodyData, ok := args["body"]; ok {
			bodyJSON, err := json.Marshal(bodyData)
			if err != nil {
				return errorResult("marshal body: " + err.Error()), nil
			}
			body = bytes.NewReader(bodyJSON)
		}
		
		httpReq, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, body)
		if err != nil {
			return errorResult("create request: " + err.Error()), nil
		}
		
		// Render and inject credential headers.
		renderedHeaders, err := c.renderer.Render(ctx, "", c.headers)
		if err != nil {
			return errorResult("render credentials: " + err.Error()), nil
		}
		for k, v := range renderedHeaders {
			httpReq.Header.Set(k, v)
		}
		
		if body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		
		resp, err := c.client.Do(httpReq)
		if err != nil {
			return errorResult("request failed: " + err.Error()), nil
		}
		defer resp.Body.Close()
		
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errorResult("read response: " + err.Error()), nil
		}
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(respBody)),
			}},
		}, nil
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
