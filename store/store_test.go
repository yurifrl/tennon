package store

import (
	"context"
	"testing"
	"time"

	"github.com/yurifrl/tennon/domain"
)

func runStoreContract(t *testing.T, s domain.Store) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.Get(ctx, domain.KindTennonConnector, "tennon", "nope"); err != domain.ErrNotFound {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}

	r := &domain.Resource{
		Kind: domain.KindTennonConnector,
		Meta: domain.ObjectMeta{Name: "github", Namespace: "tennon", Labels: map[string]string{"tags": "code"}},
		Spec: []byte(`{"transport":"streamable-http"}`),
	}
	stored, err := s.Apply(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Meta.Labels["tags"] != "code" {
		t.Fatalf("labels not persisted: %+v", stored.Meta.Labels)
	}

	// Status update must not clobber spec; Apply must not clobber status.
	stat := &domain.Resource{Kind: r.Kind, Meta: r.Meta, Status: []byte(`{"phase":"Ready","discoveredTools":5}`)}
	if err := s.UpdateStatus(ctx, stat); err != nil {
		t.Fatal(err)
	}
	r.Spec = []byte(`{"transport":"streamable-http","description":"updated"}`)
	if _, err := s.Apply(ctx, r); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Status) != `{"phase":"Ready","discoveredTools":5}` {
		t.Fatalf("Apply clobbered status: %s", got.Status)
	}
	if string(got.Spec) != `{"transport":"streamable-http","description":"updated"}` {
		t.Fatalf("spec not updated: %s", got.Spec)
	}

	list, err := s.List(ctx, domain.KindTennonConnector, "")
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %d items, %v", len(list), err)
	}

	if err := s.Delete(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name); err != domain.ErrNotFound {
		t.Fatalf("Get after delete = %v", err)
	}
}

func TestMemoryStoreContract(t *testing.T) {
	runStoreContract(t, NewMemory())
}

func TestSQLiteStoreContract(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	runStoreContract(t, s)
}

func TestWatchEmitsEvents(t *testing.T) {
	s := NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, stop, err := s.Watch(ctx, domain.KindTennonConnector)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	go func() {
		_, _ = s.Apply(context.Background(), &domain.Resource{
			Kind: domain.KindTennonConnector,
			Meta: domain.ObjectMeta{Name: "w", Namespace: "tennon"},
			Spec: []byte(`{}`),
		})
	}()

	select {
	case ev := <-events:
		if ev.Type != domain.WatchAdded || ev.Resource.Meta.Name != "w" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
}
