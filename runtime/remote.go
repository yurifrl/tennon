// Package runtime provides domain.Runtime implementations: it makes a
// connector's upstream MCP server reachable. Remote returns the upstream URL
// directly; Docker starts a container and returns its published address.
package runtime

import (
	"context"
	"fmt"

	"github.com/yurifrl/tennon/domain"
)

// Remote handles connectors whose upstream is an already-running MCP server at
// a fixed URL. There is nothing to spawn; Ensure just reports the URL.
type Remote struct{}

func NewRemote() *Remote { return &Remote{} }

func (Remote) Supports(kind string) bool { return kind == "remote" }

func (Remote) Ensure(_ context.Context, c *domain.Connector, _ map[string]string) (*domain.Endpoint, error) {
	if c.Spec.Runtime.Remote == nil || c.Spec.Runtime.Remote.URL == "" {
		return nil, fmt.Errorf("runtime/remote: connector %q has no remote.url", c.Meta.Name)
	}
	transport := c.Spec.Transport
	if transport == "" {
		transport = "streamable-http"
	}
	return &domain.Endpoint{URL: c.Spec.Runtime.Remote.URL, Transport: transport}, nil
}

func (Remote) Teardown(context.Context, *domain.Connector) error { return nil }
