package specconn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
)

// GraphQLConnector compiles a GraphQL schema into tool handlers via introspection.
type GraphQLConnector struct {
	name     string
	endpoint string
	client   *http.Client
	renderer domain.SecretRenderer
	headers  map[string]string
	schema   *graphqlSchema
}

type graphqlSchema struct {
	QueryType    *graphqlType
	MutationType *graphqlType
}

type graphqlType struct {
	Name   string
	Fields []graphqlField
}

type graphqlField struct {
	Name        string
	Description string
	Args        []graphqlArg
}

type graphqlArg struct {
	Name string
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

// NewGraphQLConnector introspects a GraphQL endpoint and returns a connector
// ready to compile tools.
func NewGraphQLConnector(ctx context.Context, name string, endpoint string, headers map[string]string, renderer domain.SecretRenderer) (*GraphQLConnector, error) {
	c := &GraphQLConnector{
		name:     name,
		endpoint: endpoint,
		client:   &http.Client{},
		renderer: renderer,
		headers:  headers,
	}
	
	// Perform introspection to get schema.
	schema, err := c.introspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("introspect GraphQL schema: %w", err)
	}
	c.schema = schema
	
	return c, nil
}

// introspect queries the GraphQL endpoint for its schema.
func (c *GraphQLConnector) introspect(ctx context.Context) (*graphqlSchema, error) {
	query := `{
		__schema {
			queryType { name fields { name description args { name type { name } } } }
			mutationType { name fields { name description args { name type { name } } } }
		}
	}`
	
	result, err := c.execute(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	
	// Parse introspection result.
	var intro struct {
		Data struct {
			Schema struct {
				QueryType    *graphqlType `json:"queryType"`
				MutationType *graphqlType `json:"mutationType"`
			} `json:"__schema"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal([]byte(result), &intro); err != nil {
		return nil, fmt.Errorf("parse introspection: %w", err)
	}
	
	return &graphqlSchema{
		QueryType:    intro.Data.Schema.QueryType,
		MutationType: intro.Data.Schema.MutationType,
	}, nil
}

// CompileTools converts GraphQL queries and mutations into domain.Tool definitions.
func (c *GraphQLConnector) CompileTools(toolsSpec domain.ToolsSpec) ([]domain.Tool, map[string]mcp.ToolHandler, error) {
	var tools []domain.Tool
	handlers := make(map[string]mcp.ToolHandler)
	
	// Add query fields as tools.
	if c.schema.QueryType != nil {
		for _, field := range c.schema.QueryType.Fields {
			toolName := field.Name
			if !toolsSpec.Permits(toolName) {
				continue
			}
			exposedName := toolsSpec.ExposedName(toolName)
			
			inputSchema := c.buildInputSchema(field)
			
			tools = append(tools, domain.Tool{
				Name:        exposedName,
				Upstream:    toolName,
				Description: field.Description,
				InputSchema: inputSchema,
				Connector:   c.name,
			})
			
			handlers[exposedName] = c.makeHandler(field.Name, "query")
		}
	}
	
	// Add mutation fields as tools.
	if c.schema.MutationType != nil {
		for _, field := range c.schema.MutationType.Fields {
			toolName := field.Name
			if !toolsSpec.Permits(toolName) {
				continue
			}
			exposedName := toolsSpec.ExposedName(toolName)
			
			inputSchema := c.buildInputSchema(field)
			
			tools = append(tools, domain.Tool{
				Name:        exposedName,
				Upstream:    toolName,
				Description: field.Description,
				InputSchema: inputSchema,
				Connector:   c.name,
			})
			
			handlers[exposedName] = c.makeHandler(field.Name, "mutation")
		}
	}
	
	return tools, handlers, nil
}

// buildInputSchema constructs a JSON schema from field arguments.
func (c *GraphQLConnector) buildInputSchema(field graphqlField) json.RawMessage {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}
	
	props := schema["properties"].(map[string]any)
	for _, arg := range field.Args {
		props[arg.Name] = map[string]any{
			"type": "string",
		}
	}
	
	result, _ := json.Marshal(schema)
	return result
}

// makeHandler returns a tool handler that executes a GraphQL operation.
func (c *GraphQLConnector) makeHandler(fieldName string, opType string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errorResult("invalid arguments: " + err.Error()), nil
			}
		}
		
		// Build GraphQL query/mutation.
		query := fmt.Sprintf("%s { %s", opType, fieldName)
		if len(args) > 0 {
			argsJSON, _ := json.Marshal(args)
			query += fmt.Sprintf("(args: %s)", string(argsJSON))
		}
		query += " }"
		
		result, err := c.execute(ctx, query, nil)
		if err != nil {
			return errorResult("execute GraphQL: " + err.Error()), nil
		}
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}
}

// execute sends a GraphQL request with rendered credentials.
func (c *GraphQLConnector) execute(ctx context.Context, query string, variables map[string]any) (string, error) {
	body := map[string]any{"query": query}
	if variables != nil {
		body["variables"] = variables
	}
	
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", err
	}
	
	// Render and inject credential headers.
	renderedHeaders, err := c.renderer.Render(ctx, "", c.headers)
	if err != nil {
		return "", fmt.Errorf("render credentials: %w", err)
	}
	for k, v := range renderedHeaders {
		httpReq.Header.Set(k, v)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return string(respBody), nil
}
