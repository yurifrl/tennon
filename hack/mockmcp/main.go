// Command mockmcp is a tiny MCP server used for local smoke-testing the Tennon
// proxy: it exposes get_status and delete_all over streamable-HTTP.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	addr := flag.String("addr", ":9090", "listen address")
	flag.Parse()

	srv := mcp.NewServer(&mcp.Implementation{Name: "mockmcp", Version: "1.0.0"}, nil)
	schema := json.RawMessage(`{"type":"object","properties":{}}`)
	reply := func(text string) mcp.ToolHandler {
		return func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
		}
	}
	srv.AddTool(&mcp.Tool{Name: "get_status", Description: "read status", InputSchema: schema}, reply("status: ok"))
	srv.AddTool(&mcp.Tool{Name: "delete_all", Description: "danger", InputSchema: schema}, reply("deleted"))

	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	log.Printf("mockmcp listening on %s/mcp", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
