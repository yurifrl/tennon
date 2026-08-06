package domain

// Phase is the lifecycle phase reported in a connector's status.
type Phase string

const (
	PhasePending  Phase = "Pending"
	PhaseReady    Phase = "Ready"
	PhaseDegraded Phase = "Degraded"
	PhaseFailed   Phase = "Failed"
)

// Connector is a registered upstream that Tennon credentials, runs, and turns
// into a set of callable tools, served over MCP at /mcp/<name>. The Kind field
// distinguishes TennonConnector, OpenApiConnector and GraphqlConnector; Phase 1
// implements TennonConnector.
type Connector struct {
	Kind   string          `json:"kind"`
	Meta   ObjectMeta      `json:"metadata"`
	Spec   ConnectorSpec   `json:"spec"`
	Status ConnectorStatus `json:"status"`
}

// ConnectorSpec is the desired state of a connector.
type ConnectorSpec struct {
	Description string `json:"description,omitempty"`

	// Runtime selects how a TennonConnector runs (exactly one of the fields set).
	Runtime   RuntimeSpec `json:"runtime,omitempty"`
	Transport string      `json:"transport,omitempty"`

	// Tools scopes and renames the exposed tool set.
	Tools ToolsSpec `json:"tools,omitempty"`
	Tags  []string  `json:"tags,omitempty"`

	// Credential injection. Header/env values may contain {{secret:name.key}}
	// templates resolved by the SecretRenderer at call time. The rendered value
	// never leaves the proxy.
	Headers map[string]string `json:"headers,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// CapabilityProfile references a CapabilityProfile (container runtime only).
	CapabilityProfile string `json:"capabilityProfile,omitempty"`

	// Spec-connector fields (OpenApiConnector / GraphqlConnector).
	SpecURL  string `json:"specUrl,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// RuntimeSpec selects where a TennonConnector's upstream MCP server runs.
// Exactly one field is set.
type RuntimeSpec struct {
	Container *ContainerRuntime `json:"container,omitempty"`
	Stdio     *StdioRuntime     `json:"stdio,omitempty"`
	Remote    *RemoteRuntime    `json:"remote,omitempty"`
	Host      *HostRuntime      `json:"host,omitempty"`
}

type ContainerRuntime struct {
	Image string `json:"image"`
}

type StdioRuntime struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type RemoteRuntime struct {
	URL string `json:"url"`
}

type HostRuntime struct {
	Address string `json:"address"`
}

// ToolsSpec controls which upstream tools are exposed and how they are named.
type ToolsSpec struct {
	ExposePrefix string   `json:"exposePrefix,omitempty"`
	Allow        []string `json:"allow,omitempty"`
	Deny         []string `json:"deny,omitempty"`
}

// ConnectorStatus is the observed state of a connector.
type ConnectorStatus struct {
	Phase           Phase  `json:"phase,omitempty"`
	DiscoveredTools int    `json:"discoveredTools,omitempty"`
	GeneratedTools  int    `json:"generatedTools,omitempty"`
	SpecHash        string `json:"specHash,omitempty"`
	URL             string `json:"url,omitempty"`
	Message         string `json:"message,omitempty"`
}

// RuntimeKind reports which runtime the spec selected, or "" if none.
func (s ConnectorSpec) RuntimeKind() string {
	switch {
	case s.Runtime.Remote != nil:
		return "remote"
	case s.Runtime.Container != nil:
		return "container"
	case s.Runtime.Stdio != nil:
		return "stdio"
	case s.Runtime.Host != nil:
		return "host"
	default:
		return ""
	}
}
