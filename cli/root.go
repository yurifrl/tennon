package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/tennon/domain"
)

// NewRootCmd builds the tctl command tree.
func NewRootCmd() *cobra.Command {
	var (
		apiURL string
		token  string
	)
	root := &cobra.Command{
		Use:   "tctl",
		Short: "Tennon control CLI — a client over the Tennon API",
	}
	root.PersistentFlags().StringVar(&apiURL, "api", envOr("TENNON_API", "http://localhost:8080"), "Tennon API base URL")
	root.PersistentFlags().StringVar(&token, "token", os.Getenv("TENNON_TOKEN"), "bearer token")

	client := func() *Client { return NewClient(strings.TrimRight(apiURL, "/"), token) }

	root.AddCommand(newConnectorCmd(client))
	root.AddCommand(newSecretCmd(client))
	root.AddCommand(newMCPCmd(client))
	root.AddCommand(newExecutionCmd(client))
	return root
}

func newConnectorCmd(client func() *Client) *cobra.Command {
	cmd := &cobra.Command{Use: "connector", Short: "Manage connectors"}

	var (
		remote, container, stdioCmd string
		transport, exposePrefix     string
		headers, allow, deny, tags  []string
		description                 string
		jsonOut                     bool
	)
	add := &cobra.Command{
		Use:   "add <name>",
		Short: "Register a connector",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := domain.Connector{
				Kind: domain.KindTennonConnector,
				Meta: domain.ObjectMeta{Name: args[0], Labels: tagLabels(tags)},
				Spec: domain.ConnectorSpec{
					Description: description,
					Transport:   defaultStr(transport, "streamable-http"),
					Headers:     kvMap(headers),
					Tags:        tags,
					Tools: domain.ToolsSpec{
						ExposePrefix: exposePrefix,
						Allow:        splitCSV(allow),
						Deny:         splitCSV(deny),
					},
				},
			}
			switch {
			case remote != "":
				c.Spec.Runtime.Remote = &domain.RemoteRuntime{URL: remote}
			case container != "":
				c.Spec.Runtime.Container = &domain.ContainerRuntime{Image: container}
			case stdioCmd != "":
				parts := strings.Fields(stdioCmd)
				c.Spec.Runtime.Stdio = &domain.StdioRuntime{Command: parts[0], Args: parts[1:]}
			default:
				return fmt.Errorf("one of --remote, --container, or --stdio is required")
			}
			var out domain.Connector
			if err := client().do(cmd.Context(), "POST", "/api/v1/connectors", c, &out); err != nil {
				return err
			}
			return print(jsonOut, out, fmt.Sprintf("connector %q created (%s)", out.Meta.Name, out.Spec.RuntimeKind()))
		},
	}
	add.Flags().StringVar(&remote, "remote", "", "remote MCP server URL")
	add.Flags().StringVar(&container, "container", "", "container image")
	add.Flags().StringVar(&stdioCmd, "stdio", "", "stdio command line")
	add.Flags().StringVar(&transport, "transport", "streamable-http", "transport")
	add.Flags().StringVar(&exposePrefix, "expose-prefix", "", "prefix applied to exposed tool names")
	add.Flags().StringArrayVar(&headers, "header", nil, "credential header, key=value (may use {{secret:name.key}})")
	add.Flags().StringSliceVar(&allow, "allow", nil, "allowed tool globs")
	add.Flags().StringSliceVar(&deny, "deny", nil, "denied tool globs")
	add.Flags().StringSliceVar(&tags, "tag", nil, "tags")
	add.Flags().StringVar(&description, "description", "", "agent-visible description")
	add.Flags().BoolVar(&jsonOut, "json", false, "JSON output")

	var listJSON bool
	list := &cobra.Command{
		Use:   "list",
		Short: "List connectors",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var resp struct {
				Items []domain.Connector `json:"items"`
			}
			if err := client().do(cmd.Context(), "GET", "/api/v1/connectors", nil, &resp); err != nil {
				return err
			}
			if listJSON {
				return print(true, resp.Items, "")
			}
			for _, c := range resp.Items {
				fmt.Printf("%-24s %-10s %-16s %s\n", c.Meta.Name, c.Spec.RuntimeKind(), c.Status.Phase, c.Status.URL)
			}
			return nil
		},
	}
	list.Flags().BoolVar(&listJSON, "json", false, "JSON output")

	describe := &cobra.Command{
		Use:   "describe <name>",
		Short: "Show a connector",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var c domain.Connector
			if err := client().do(cmd.Context(), "GET", "/api/v1/connectors/"+args[0], nil, &c); err != nil {
				return err
			}
			return print(true, c, "")
		},
	}

	del := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a connector",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client().do(cmd.Context(), "DELETE", "/api/v1/connectors/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Printf("connector %q deleted\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(add, list, describe, del)
	return cmd
}

func newSecretCmd(client func() *Client) *cobra.Command {
	cmd := &cobra.Command{Use: "secret", Short: "Manage secrets"}
	var fromLiteral []string
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a secret from literals",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data := kvMap(fromLiteral)
			if len(data) == 0 {
				return fmt.Errorf("at least one --from-literal key=value is required")
			}
			body := map[string]any{
				"metadata": map[string]string{"name": args[0]},
				"data":     data,
			}
			if err := client().do(cmd.Context(), "POST", "/api/v1/secrets", body, nil); err != nil {
				return err
			}
			fmt.Printf("secret %q created (%d keys)\n", args[0], len(data))
			return nil
		},
	}
	create.Flags().StringArrayVar(&fromLiteral, "from-literal", nil, "key=value (repeatable)")
	cmd.AddCommand(create)
	return cmd
}

func kvMap(pairs []string) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	m := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if ok {
			m[k] = v
		}
	}
	return m
}

func splitCSV(in []string) []string {
	var out []string
	for _, s := range in {
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func tagLabels(tags []string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	return map[string]string{"tags": strings.Join(tags, ",")}
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func print(jsonOut bool, v any, human string) error {
	if jsonOut {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	if human != "" {
		fmt.Println(human)
	}
	return nil
}

func newExecutionCmd(client func() *Client) *cobra.Command {
	cmd := &cobra.Command{Use: "execution", Short: "Manage executions", Aliases: []string{"exec"}}

	var listJSON bool
	var listState string
	list := &cobra.Command{
		Use:   "list",
		Short: "List executions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var resp struct {
				Items []domain.Execution `json:"items"`
			}
			url := "/api/v1/executions"
			if listState != "" {
				url += "?state=" + listState
			}
			if err := client().do(cmd.Context(), "GET", url, nil, &resp); err != nil {
				return err
			}
			if listJSON {
				return print(true, resp.Items, "")
			}
			for _, e := range resp.Items {
				fmt.Printf("%-24s %-12s %-20s %-20s %s\n", e.Meta.Name, e.Status.State, e.Spec.Connector, e.Spec.Tool, e.Spec.Subject)
			}
			return nil
		},
	}
	list.Flags().BoolVar(&listJSON, "json", false, "JSON output")
	list.Flags().StringVar(&listState, "state", "", "filter by state (pending, approved, completed, denied, failed)")

	resume := &cobra.Command{
		Use:   "resume <execution-id>",
		Short: "Approve and resume a pending execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client().do(cmd.Context(), "POST", "/api/v1/executions/"+args[0]+"/approve", nil, nil); err != nil {
				return err
			}
			fmt.Printf("execution %q approved and resumed\n", args[0])
			return nil
		},
	}

	deny := &cobra.Command{
		Use:   "deny <execution-id>",
		Short: "Deny a pending execution",
			Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client().do(cmd.Context(), "POST", "/api/v1/executions/"+args[0]+"/deny", nil, nil); err != nil {
				return err
			}
			fmt.Printf("execution %q denied\n", args[0])
			return nil
		},
	}

	describe := &cobra.Command{
		Use:   "describe <execution-id>",
		Short: "Show an execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var e domain.Execution
			if err := client().do(cmd.Context(), "GET", "/api/v1/executions/"+args[0], nil, &e); err != nil {
				return err
			}
			return print(true, e, "")
		},
	}

	cmd.AddCommand(list, resume, deny, describe)
	return cmd
}
