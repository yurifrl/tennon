package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurifrl/tennon/domain"
)

// NewMCPServer creates an MCP server that exposes CLI operations as tools,
// bridging to the Tennon API via the provided client.
func NewMCPServer(apiURL, token string) *mcp.Server {
	client := NewClient(apiURL, token)
	srv := mcp.NewServer(&mcp.Implementation{Name: "tennon-cli", Version: "1.0.0"}, nil)

	// tennon_connector_list: List connectors
	schema := json.RawMessage(`{"type":"object","properties":{}}`)
	srv.AddTool(&mcp.Tool{
		Name:        "tennon_connector_list",
		Description: "List all Tennon connectors",
		InputSchema: schema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var resp struct {
			Items []domain.Connector `json:"items"`
		}
		if err := client.do(ctx, "GET", "/api/v1/connectors", nil, &resp); err != nil {
			return nil, err
		}
		data, err := json.Marshal(resp.Items)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil
	})

	// tennon_connector_create: Create a connector
	createSchema := json.RawMessage(`{
		"type":"object",
		"required":["name"],
		"properties":{
			"name":{"type":"string","description":"Connector name"},
			"description":{"type":"string","description":"Connector description"},
			"remote":{"type":"string","description":"Remote MCP server URL"},
			"container":{"type":"string","description":"Container image"},
			"stdio":{"type":"string","description":"Stdio command line"},
			"transport":{"type":"string","description":"Transport type","enum":["stdio","streamable-http"]},
			"expose_prefix":{"type":"string","description":"Prefix for exposed tool names"},
			"allow":{"type":"array","items":{"type":"string"},"description":"Allowed tool globs"},
			"deny":{"type":"array","items":{"type":"string"},"description":"Denied tool globs"},
			"tags":{"type":"array","items":{"type":"string"},"description":"Tags"}
		}
	}`)
	srv.AddTool(&mcp.Tool{
		Name:        "tennon_connector_create",
		Description: "Create a new Tennon connector",
		InputSchema: createSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse arguments
		var args struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			Remote       string   `json:"remote"`
			Container    string   `json:"container"`
			Stdio        string   `json:"stdio"`
			Transport    string   `json:"transport"`
			ExposePrefix string   `json:"expose_prefix"`
			Allow        []string `json:"allow"`
			Deny         []string `json:"deny"`
			Tags         []string `json:"tags"`
		}
		argBytes, _ := json.Marshal(req.Params.Arguments)
		if err := json.Unmarshal(argBytes, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		// Build connector spec
		c := domain.Connector{
			Kind: domain.KindTennonConnector,
			Meta: domain.ObjectMeta{Name: args.Name, Labels: tagLabels(args.Tags)},
			Spec: domain.ConnectorSpec{
				Description: args.Description,
				Transport:   defaultStr(args.Transport, "streamable-http"),
				Tags:        args.Tags,
				Tools: domain.ToolsSpec{
					ExposePrefix: args.ExposePrefix,
					Allow:        args.Allow,
					Deny:         args.Deny,
				},
			},
		}
		if args.Remote != "" {
			c.Spec.Runtime.Remote = &domain.RemoteRuntime{URL: args.Remote}
		}
		if args.Container != "" {
			c.Spec.Runtime.Container = &domain.ContainerRuntime{Image: args.Container}
		}
		if args.Stdio != "" {
			c.Spec.Runtime.Stdio = &domain.StdioRuntime{Command: args.Stdio}
		}

		var out domain.Connector
		if err := client.do(ctx, "POST", "/api/v1/connectors", c, &out); err != nil {
			return nil, err
		}

		data, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil
	})

	// tennon_secret_create: Create secrets
	secretSchema := json.RawMessage(`{
		"type":"object",
		"required":["name","data"],
		"properties":{
			"name":{"type":"string","description":"Secret name"},
			"data":{"type":"object","description":"Key-value pairs"}
		}
	}`)
	srv.AddTool(&mcp.Tool{
		Name:        "tennon_secret_create",
		Description: "Create a Tennon secret",
		InputSchema: secretSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name string            `json:"name"`
			Data map[string]string `json:"data"`
		}
		argBytes, _ := json.Marshal(req.Params.Arguments)
		if err := json.Unmarshal(argBytes, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}

		body := map[string]any{
			"metadata": map[string]string{"name": args.Name},
			"data":     args.Data,
		}
		if err := client.do(ctx, "POST", "/api/v1/secrets", body, nil); err != nil {
			return nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("Secret %q created with %d keys", args.Name, len(args.Data)),
			}},
		}, nil
	})

	// tennon_tool_call: Call tools through Tennon (not yet available)
	srv.AddTool(&mcp.Tool{
		Name:        "tennon_tool_call",
		Description: "Call a tool through Tennon (not yet available)",
		InputSchema: schema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: `{"error":"tennon_tool_call is not yet available - API endpoint not implemented"}`,
			}},
			IsError: true,
		}, nil
	})

	// tennon_policy_manage: Manage access policies (not yet available)
	srv.AddTool(&mcp.Tool{
		Name:        "tennon_policy_manage",
		Description: "Manage Tennon access policies (not yet available)",
		InputSchema: schema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: `{"error":"tennon_policy_manage is not yet available - API endpoint not implemented"}`,
			}},
			IsError: true,
		}, nil
	})

	return srv
}
