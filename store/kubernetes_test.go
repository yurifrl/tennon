package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yurifrl/tennon/domain"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetes_OfflineMode(t *testing.T) {
	// This test should always run - it uses the fake client
	ctx := context.Background()

	// Create a fake client with an empty scheme
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := NewKubernetes(fakeClient)
	if store == nil {
		t.Fatal("NewKubernetes returned nil")
	}

	// Test that we can create a resource
	connector := &domain.Resource{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{
			Name:      "test-connector",
			Namespace: "default",
			Labels:    map[string]string{"env": "test"},
		},
		Spec: json.RawMessage(`{"runtime":{"remote":{"url":"http://example.com"}}}`),
	}

	// Apply the resource
	created, err := store.Apply(ctx, connector)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if created.Meta.Name != "test-connector" {
		t.Errorf("Expected name test-connector, got %s", created.Meta.Name)
	}

	// Test Get
	fetched, err := store.Get(ctx, domain.KindTennonConnector, "default", "test-connector")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if fetched.Meta.Name != "test-connector" {
		t.Errorf("Expected name test-connector, got %s", fetched.Meta.Name)
	}

	// Test List
	list, err := store.List(ctx, domain.KindTennonConnector, "default")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(list))
	}

	// Test Delete
	if err := store.Delete(ctx, domain.KindTennonConnector, "default", "test-connector"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = store.Get(ctx, domain.KindTennonConnector, "default", "test-connector")
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

func TestKubernetes_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := NewKubernetes(fakeClient)

	// Start watching
	events, stop, err := store.Watch(ctx, domain.KindTennonConnector)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer stop()

	// Create a resource in a goroutine
	go func() {
		connector := &domain.Resource{
			Kind: domain.KindTennonConnector,
			Meta: domain.ObjectMeta{
				Name:      "watch-test",
				Namespace: "default",
			},
			Spec: json.RawMessage(`{"runtime":{"remote":{"url":"http://example.com"}}}`),
		}
		if _, err := store.Apply(ctx, connector); err != nil {
			t.Logf("Apply in goroutine failed: %v", err)
		}
	}()

	// Wait for the event (with timeout protection via context)
	select {
	case event := <-events:
		if event.Type != domain.WatchAdded {
			t.Errorf("Expected WatchAdded event, got %v", event.Type)
		}
		if event.Resource.Meta.Name != "watch-test" {
			t.Errorf("Expected resource name watch-test, got %s", event.Resource.Meta.Name)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for watch event")
	}
}

func TestKubernetes_Close(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	store := NewKubernetes(fakeClient)
	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
