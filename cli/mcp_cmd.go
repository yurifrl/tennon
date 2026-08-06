package cli

import (
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// newMCPCmd builds the 'tctl mcp' command tree.
func newMCPCmd(client func() *Client) *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "MCP server operations"}

	var (
		transport string
		port      int
	)
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the embedded MCP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := client()
			srv := NewMCPServer(c.BaseURL, c.Token)

			switch transport {
			case "stdio":
				log.Printf("Starting MCP server on stdio...")
				return srv.Run(cmd.Context(), &mcp.StdioTransport{})

			case "streamable-http":
				addr := fmt.Sprintf(":%d", port)
				log.Printf("Starting MCP server on http://localhost%s/mcp", addr)
				h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
				return http.ListenAndServe(addr, h)

			default:
				return fmt.Errorf("unknown transport %q (must be stdio or streamable-http)", transport)
			}
		},
	}
	serve.Flags().StringVar(&transport, "transport", "stdio", "transport type: stdio or streamable-http")
	serve.Flags().IntVar(&port, "port", 8080, "port for streamable-http transport")

	cmd.AddCommand(serve)
	return cmd
}
