package runtime_test

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yurifrl/tennon/auth"
	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/proxy"
	"github.com/yurifrl/tennon/runtime"
	"github.com/yurifrl/tennon/secret"
	"github.com/yurifrl/tennon/store"
)

func TestDockerRuntimeE2E(t *testing.T) {
	// Skip unless TENNON_TEST_DOCKER=1 is set.
	if os.Getenv("TENNON_TEST_DOCKER") != "1" {
		t.Skip("set TENNON_TEST_DOCKER=1 to run docker e2e tests")
	}

	// Check if docker is available.
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker not available:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build the mockmcp test image.
	imageName := "tennon-test-mockmcp:latest"
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, "-f", "hack/mockmcp/Dockerfile", ".")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Skipf("failed to build test image: %v\n%s", err, out)
	}

	// Ensure cleanup even if test fails.
	defer func() {
		_ = exec.Command("docker", "rm", "-f", "tennon-mockmcp-e2e").Run()
	}()

	// 1. Store with no secrets (mockmcp doesn't need auth).
	st := store.NewMemory()

	// 2. Proxy wired with docker runtime.
	dockerRT := runtime.NewDocker()
	px := proxy.New(
		secret.NewRenderer(st, "tennon"),
		runtime.NewMulti(dockerRT),
		auth.NewAllowAllAuthorizer(),
		auth.NewAnonymousVerifier(),
		nil,
	)

	// 3. Register a connector using the docker runtime.
	conn := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{
			Name:      "mockmcp-e2e",
			Namespace: "tennon",
			Labels:    map[string]string{"tennon.io/port": "8080"},
		},
		Spec: domain.ConnectorSpec{
			Runtime:   domain.RuntimeSpec{Container: &domain.ContainerRuntime{Image: imageName}},
			Transport: "streamable-http",
			Tools:     domain.ToolsSpec{ExposePrefix: "test_", Allow: []string{"*"}},
		},
	}
	tools, err := px.Register(ctx, conn)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// mockmcp exposes get_status and delete_all.
	if len(tools) != 2 {
		t.Fatalf("exposed tools = %+v, want 2 tools", tools)
	}

	// Give the container a moment to start and listen.
	time.Sleep(2 * time.Second)

	// 4. Connect a client to the proxy and call a tool.
	proxySrv := httptest.NewServer(px.Handler())
	defer proxySrv.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-agent", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             proxySrv.URL + "/mcp/mockmcp-e2e",
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// 5. List tools.
	listed, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(listed.Tools) != 2 {
		t.Fatalf("client sees %d tools, want 2", len(listed.Tools))
	}

	// 6. Call test_get_status through the proxy.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "test_get_status"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("tool error: %+v", res)
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok || tc.Text != "status: ok" {
		t.Fatalf("unexpected result: %+v", res.Content[0])
	}

	// 7. Teardown.
	px.Deregister(ctx, conn)
}
