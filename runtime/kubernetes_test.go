package runtime

import (
	"context"
	"testing"

	"github.com/yurifrl/tennon/domain"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetes_OfflineMode(t *testing.T) {
	// This test should always run - it uses the fake client
	ctx := context.Background()

	// Create a fake client with core v1 scheme
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sRuntime := NewKubernetes(fakeClient, "test-namespace")

	if k8sRuntime == nil {
		t.Fatal("NewKubernetes returned nil")
	}

	// Test Supports
	if !k8sRuntime.Supports("container") {
		t.Error("Expected to support 'container' runtime kind")
	}

	if k8sRuntime.Supports("remote") {
		t.Error("Should not support 'remote' runtime kind")
	}

	// Create a test connector
	connector := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{
			Name:      "test-connector",
			Namespace: "test-namespace",
			Labels:    map[string]string{"tennon.io/port": "9000"},
		},
		Spec: domain.ConnectorSpec{
			Runtime: domain.RuntimeSpec{
				Container: &domain.ContainerRuntime{
					Image: "ghcr.io/test/test-mcp:latest",
				},
			},
			Transport: "streamable-http",
			Env: map[string]string{
				"TEST_VAR": "test-value",
			},
		},
	}

	// Test Ensure
	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}

	endpoint, err := k8sRuntime.Ensure(ctx, connector, headers)
	if err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}

	if endpoint == nil {
		t.Fatal("Expected endpoint, got nil")
	}

	if endpoint.Transport != "streamable-http" {
		t.Errorf("Expected transport streamable-http, got %s", endpoint.Transport)
	}

	// Verify URL format
	expectedAddress := "tennon-test-connector.test-namespace.svc.cluster.local:9000"
	expectedURL := "http://" + expectedAddress + "/mcp"
	if endpoint.URL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, endpoint.URL)
	}

	// Test Teardown
	if err := k8sRuntime.Teardown(ctx, connector); err != nil {
		t.Fatalf("Teardown failed: %v", err)
	}
}

func TestKubernetes_NoContainer(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sRuntime := NewKubernetes(fakeClient, "test-namespace")

	// Connector without container spec should fail
	connector := &domain.Connector{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{
			Name:      "no-container",
			Namespace: "test-namespace",
		},
		Spec: domain.ConnectorSpec{
			Runtime: domain.RuntimeSpec{
				Remote: &domain.RemoteRuntime{
					URL: "http://example.com",
				},
			},
		},
	}

	_, err := k8sRuntime.Ensure(ctx, connector, nil)
	if err == nil {
		t.Error("Expected error for connector without container spec")
	}
}

func TestKubernetes_DefaultNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test with empty namespace - should default to "tennon"
	k8sRuntime := NewKubernetes(fakeClient, "")
	if k8sRuntime.namespace != "tennon" {
		t.Errorf("Expected default namespace 'tennon', got %s", k8sRuntime.namespace)
	}

	// Test with explicit namespace
	k8sRuntime2 := NewKubernetes(fakeClient, "custom-ns")
	if k8sRuntime2.namespace != "custom-ns" {
		t.Errorf("Expected namespace 'custom-ns', got %s", k8sRuntime2.namespace)
	}
}
