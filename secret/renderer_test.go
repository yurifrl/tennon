package secret

import (
	"context"
	"testing"

	"github.com/yurifrl/tennon/domain"
	"github.com/yurifrl/tennon/store"
)

func TestRenderString(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory()
	res, _ := domain.NewSecretResource("tennon", "grafana-creds", map[string]string{"token": "s3cr3t"})
	if _, err := s.Apply(ctx, res); err != nil {
		t.Fatal(err)
	}
	r := NewRenderer(s, "tennon")

	got, err := r.RenderString(ctx, "tennon", "Bearer {{secret:grafana-creds.token}}")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Bearer s3cr3t" {
		t.Fatalf("RenderString = %q", got)
	}

	// No template => passthrough.
	if got, _ := r.RenderString(ctx, "tennon", "static"); got != "static" {
		t.Fatalf("passthrough = %q", got)
	}

	// Missing key is an error.
	if _, err := r.RenderString(ctx, "tennon", "{{secret:grafana-creds.missing}}"); err == nil {
		t.Fatal("expected error for missing key")
	}

	// Missing secret is an error.
	if _, err := r.RenderString(ctx, "tennon", "{{secret:nope.token}}"); err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestRenderMap(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory()
	res, _ := domain.NewSecretResource("tennon", "g", map[string]string{"token": "abc"})
	_, _ = s.Apply(ctx, res)
	r := NewRenderer(s, "tennon")

	out, err := r.Render(ctx, "tennon", map[string]string{
		"Authorization": "Bearer {{secret:g.token}}",
		"X-Static":      "v",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["Authorization"] != "Bearer abc" || out["X-Static"] != "v" {
		t.Fatalf("Render = %+v", out)
	}
}
