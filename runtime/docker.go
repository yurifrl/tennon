package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/yurifrl/tennon/domain"
)

// Docker runs container connectors as local Docker containers. It publishes the
// container's MCP port on a free host port and returns the local URL. The
// rendered credential is passed as container environment variables.
//
// The container image is expected to speak MCP over streamable-HTTP on its
// listen port (default 8080; override with the "tennon.io/port" label). stdio
// images are bridged by Supergateway in the container runtime (see
// docs/arch/08-runtime-isolation.md) — not yet wired here.
type Docker struct {
	bin  string
	mu   sync.Mutex
	ctrs map[string]containerInfo // connector name -> running container
}

type containerInfo struct {
	id       string
	hostPort int
}

func NewDocker() *Docker {
	return &Docker{bin: "docker", ctrs: map[string]containerInfo{}}
}

func (*Docker) Supports(kind string) bool { return kind == "container" }

func (d *Docker) Ensure(ctx context.Context, c *domain.Connector, headers map[string]string) (*domain.Endpoint, error) {
	if c.Spec.Runtime.Container == nil || c.Spec.Runtime.Container.Image == "" {
		return nil, fmt.Errorf("runtime/docker: connector %q has no container.image", c.Meta.Name)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	name := "tennon-" + c.Meta.Name
	if info, ok := d.ctrs[c.Meta.Name]; ok && d.running(ctx, info.id) {
		return d.endpoint(info.hostPort, c), nil
	}
	_ = d.remove(ctx, name) // clear any stale container with this name

	port, err := freePort()
	if err != nil {
		return nil, err
	}
	containerPort := "8080"
	if p := c.Meta.Labels["tennon.io/port"]; p != "" {
		containerPort = p
	}

	args := []string{"run", "-d", "--rm", "--name", name,
		"-p", fmt.Sprintf("127.0.0.1:%d:%s", port, containerPort)}
	// Rendered credential env, in a stable order.
	for _, k := range sortedKeys(headers) {
		args = append(args, "-e", k+"="+headers[k])
	}
	for _, k := range sortedKeys(c.Spec.Env) {
		args = append(args, "-e", k+"="+c.Spec.Env[k])
	}
	args = append(args, c.Spec.Runtime.Container.Image)

	out, err := d.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("runtime/docker: run %q: %w", c.Meta.Name, err)
	}
	id := strings.TrimSpace(out)
	d.ctrs[c.Meta.Name] = containerInfo{id: id, hostPort: port}
	return d.endpoint(port, c), nil
}

func (d *Docker) Teardown(ctx context.Context, c *domain.Connector) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.ctrs, c.Meta.Name)
	return d.remove(ctx, "tennon-"+c.Meta.Name)
}

func (d *Docker) endpoint(hostPort int, c *domain.Connector) *domain.Endpoint {
	transport := c.Spec.Transport
	if transport == "" {
		transport = "streamable-http"
	}
	return &domain.Endpoint{URL: fmt.Sprintf("http://127.0.0.1:%d/mcp", hostPort), Transport: transport}
}

func (d *Docker) running(ctx context.Context, id string) bool {
	out, err := d.run(ctx, "inspect", "-f", "{{.State.Running}}", id)
	return err == nil && strings.TrimSpace(out) == "true"
}

func (d *Docker) remove(ctx context.Context, name string) error {
	_, err := d.run(ctx, "rm", "-f", name)
	return err
}

func (d *Docker) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, d.bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", d.bin, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
