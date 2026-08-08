package domain

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrNotFound is returned by a Store when an object does not exist.
var ErrNotFound = errors.New("not found")

// Resource is the generic on-store envelope for every Tennon object. The store
// persists Spec and Status as opaque JSON so a single table (or the apiserver)
// can hold every kind without knowing their shapes. Typed accessors convert
// between Resource and the concrete types (e.g. Connector).
type Resource struct {
	Kind   string          `json:"kind"`
	Meta   ObjectMeta      `json:"metadata"`
	Spec   json.RawMessage `json:"spec,omitempty"`
	Status json.RawMessage `json:"status,omitempty"`
}

// WatchEventType categorises a store change.
type WatchEventType string

const (
	WatchAdded    WatchEventType = "ADDED"
	WatchModified WatchEventType = "MODIFIED"
	WatchDeleted  WatchEventType = "DELETED"
)

// WatchEvent is emitted on a Store watch channel.
type WatchEvent struct {
	Type     WatchEventType
	Resource *Resource
}

// Store persists desired-state CRD objects (and audit records) in the
// configured backend. It emulates the apiserver's get/list/watch semantics so
// the same reconcile loops run on Kubernetes, Postgres and SQLite.
type Store interface {
	Get(ctx context.Context, kind, namespace, name string) (*Resource, error)
	List(ctx context.Context, kind, namespace string) ([]*Resource, error)
	// Apply upserts desired state (Meta + Spec). Status is preserved.
	Apply(ctx context.Context, r *Resource) (*Resource, error)
	// UpdateStatus writes only the Status of an existing object.
	UpdateStatus(ctx context.Context, r *Resource) error
	Delete(ctx context.Context, kind, namespace, name string) error
	// Watch streams changes for a kind until ctx is cancelled or stop is called.
	Watch(ctx context.Context, kind string) (events <-chan WatchEvent, stop func(), err error)
	Close() error
}

// Endpoint describes how the proxy reaches a connector's upstream MCP server.
type Endpoint struct {
	URL       string // upstream MCP streamable-HTTP URL
	Transport string
}

// Runtime makes a connector's upstream reachable. Remote returns the upstream
// URL directly; Docker/Kubernetes start a workload and return its address.
type Runtime interface {
	// Ensure brings the connector's upstream up (idempotent) and returns how to
	// reach it. Headers are the already-rendered credential headers.
	Ensure(ctx context.Context, c *Connector, headers map[string]string) (*Endpoint, error)
	Teardown(ctx context.Context, c *Connector) error
	// Supports reports whether this runtime handles the connector's runtime kind.
	Supports(kind string) bool
}

// SecretRenderer resolves {{secret:name.key}} templates against secrets in the
// store. Rendered values never leave the proxy/controller.
type SecretRenderer interface {
	RenderString(ctx context.Context, namespace, template string) (string, error)
	Render(ctx context.Context, namespace string, templates map[string]string) (map[string]string, error)
}

// Authorizer decides whether a subject may make a tool call. Phase 1 ships a
// permissive default; Phase 3 compiles Policy to Cedar behind this port.
type Authorizer interface {
	Decide(ctx context.Context, sub Subject, call ToolCall) Decision
}

// IdentityVerifier resolves a bearer credential to a Subject. Phase 1 ships a
// permissive default; Phases 2/4 add OIDC, tokenless trust and fine-grained
// tokens behind this port.
type IdentityVerifier interface {
	// Verify resolves the bearer credential (may be empty) to a Subject.
	Verify(ctx context.Context, bearer string) (Subject, error)
}

// ExecutionCompleter manages pending executions and completes them when approved.
// The proxy implements this to complete paused tool calls.
type ExecutionCompleter interface {
	// ApproveExecution marks an execution as approved and completes the upstream call.
	ApproveExecution(ctx context.Context, execID string) error
	// DenyExecution marks an execution as denied.
	DenyExecution(ctx context.Context, execID string) error
	// ListExecutions returns all executions, optionally filtered by state.
	ListExecutions(ctx context.Context, state string) ([]*Execution, error)
	// GetExecution returns a single execution by id.
	GetExecution(ctx context.Context, execID string) (*Execution, error)
}
