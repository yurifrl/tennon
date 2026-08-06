package domain

import "encoding/json"

// Tool is a single callable exposed by a connector through the proxy. Name is
// the caller-facing name (with the connector's exposePrefix applied); Upstream
// is the original name on the upstream MCP server.
type Tool struct {
	Name        string          `json:"name"`
	Upstream    string          `json:"upstream"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	Connector   string          `json:"connector"`
	Tags        []string        `json:"tags,omitempty"`
}

// ToolCall is a request to invoke a tool through the proxy.
type ToolCall struct {
	Connector string         `json:"connector"`
	Tool      string         `json:"tool"` // caller-facing (prefixed) name
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Effect is the outcome of an authorization rule. The proxy honours the
// most-restrictive-wins order: block > ask > allow.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectAsk   Effect = "ask"
	EffectBlock Effect = "block"
)

// Decision is the authorizer's verdict for a single tool call.
type Decision struct {
	Effect Effect `json:"effect"`
	Reason string `json:"reason,omitempty"`
}
