package runtime

import (
	"context"
	"fmt"

	"github.com/yurifrl/tennon/domain"
)

// Multi dispatches to the sub-runtime that supports a connector's runtime kind.
type Multi struct {
	runtimes []domain.Runtime
}

func NewMulti(runtimes ...domain.Runtime) *Multi { return &Multi{runtimes: runtimes} }

func (m *Multi) pick(c *domain.Connector) (domain.Runtime, error) {
	kind := c.Spec.RuntimeKind()
	if kind == "" {
		return nil, fmt.Errorf("runtime: connector %q has no runtime configured", c.Meta.Name)
	}
	for _, r := range m.runtimes {
		if r.Supports(kind) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("runtime: no backend supports runtime kind %q (connector %q)", kind, c.Meta.Name)
}

func (m *Multi) Supports(kind string) bool {
	for _, r := range m.runtimes {
		if r.Supports(kind) {
			return true
		}
	}
	return false
}

func (m *Multi) Ensure(ctx context.Context, c *domain.Connector, headers map[string]string) (*domain.Endpoint, error) {
	r, err := m.pick(c)
	if err != nil {
		return nil, err
	}
	return r.Ensure(ctx, c, headers)
}

func (m *Multi) Teardown(ctx context.Context, c *domain.Connector) error {
	r, err := m.pick(c)
	if err != nil {
		return nil // nothing to tear down for an unconfigured runtime
	}
	return r.Teardown(ctx, c)
}

// Config selects which runtimes are enabled.
type Config struct {
	Backend string // auto | remote | docker | kubernetes
}

// NewFromConfig builds the domain.Runtime for cfg. "auto" enables remote +
// docker (local host mode).
func NewFromConfig(cfg Config) (domain.Runtime, error) {
	switch cfg.Backend {
	case "", "auto":
		return NewMulti(NewRemote(), NewDocker()), nil
	case "remote":
		return NewMulti(NewRemote()), nil
	case "docker":
		return NewMulti(NewRemote(), NewDocker()), nil
	case "kubernetes":
		// Kubernetes backend requires a live cluster; in offline mode this will fail.
		// Use NewKubernetesWithClient in tests.
		return nil, fmt.Errorf("runtime: kubernetes backend requires cluster configuration (use NewKubernetesWithClient)")
	default:
		return nil, fmt.Errorf("runtime: unknown backend %q", cfg.Backend)
	}
}
