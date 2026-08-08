package domain

import (
	"context"
	"testing"
	"time"
)

type mockTokenStore struct {
	tokens map[string]*Resource
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*Resource)}
}

func (m *mockTokenStore) Get(_ context.Context, kind, namespace, name string) (*Resource, error) {
	if kind != KindToken {
		return nil, ErrNotFound
	}
	key := namespace + "/" + name
	if r, ok := m.tokens[key]; ok {
		return r, nil
	}
	return nil, ErrNotFound
}

func (m *mockTokenStore) List(_ context.Context, kind, _ string) ([]*Resource, error) {
	if kind != KindToken {
		return nil, nil
	}
	var out []*Resource
	for _, r := range m.tokens {
		out = append(out, r)
	}
	return out, nil
}

func (m *mockTokenStore) Delete(_ context.Context, kind, namespace, name string) error {
	if kind != KindToken {
		return ErrNotFound
	}
	key := namespace + "/" + name
	delete(m.tokens, key)
	return nil
}

func (m *mockTokenStore) Apply(_ context.Context, r *Resource) (*Resource, error) {
	key := r.Meta.Namespace + "/" + r.Meta.Name
	m.tokens[key] = r
	return r, nil
}

func (m *mockTokenStore) UpdateStatus(_ context.Context, r *Resource) error {
	key := r.Meta.Namespace + "/" + r.Meta.Name
	m.tokens[key] = r
	return nil
}

func (m *mockTokenStore) Close() error {
	return nil
}

func (m *mockTokenStore) Watch(_ context.Context, _ string) (<-chan WatchEvent, func(), error) {
	return nil, nil, nil
}

func TestTokenService_Mint(t *testing.T) {
	store := newMockTokenStore()
	svc := NewTokenService(store)

	req := MintRequest{
		Name:      "test-token",
		Namespace: "default",
		Subject: Subject{
			Kind: "ServiceAccount",
			ID:   "service:ci-bot",
		},
		Grants: []TokenGrant{
			{
				Resources: []string{"grafana"},
				Tools:     []string{"query"},
				Effect:    "allow",
			},
		},
		TTL: time.Hour,
	}

	resp, err := svc.Mint(context.Background(), req)
	if err != nil {
		t.Fatalf("mint failed: %v", err)
	}

	if resp.Token == nil {
		t.Fatal("expected token in response")
	}

	if resp.SecretKey == "" {
		t.Fatal("expected secret key in response")
	}

	if resp.Token.ObjectMeta.Name != "test-token" {
		t.Errorf("expected name test-token, got %s", resp.Token.ObjectMeta.Name)
	}

	if resp.Token.Spec.Subject.ID != "service:ci-bot" {
		t.Errorf("expected subject service:ci-bot, got %s", resp.Token.Spec.Subject.ID)
	}

	if len(resp.Token.Spec.Grants) != 1 {
		t.Fatalf("expected 1 grant, got %d", len(resp.Token.Spec.Grants))
	}

	if resp.Token.Spec.Grants[0].Effect != "allow" {
		t.Errorf("expected effect allow, got %s", resp.Token.Spec.Grants[0].Effect)
	}

	// Verify secret key starts with tn_
	if len(resp.SecretKey) < 4 || resp.SecretKey[:3] != "tn_" {
		t.Errorf("expected secret key to start with tn_, got %s", resp.SecretKey[:10])
	}

	// Verify token was stored
	key := "default/test-token"
	if _, ok := store.tokens[key]; !ok {
		t.Fatal("token was not stored")
	}
}

func TestTokenService_Revoke(t *testing.T) {
	store := newMockTokenStore()
	svc := NewTokenService(store)

	// Mint a token first
	req := MintRequest{
		Name:      "test-token",
		Namespace: "default",
		Subject:   Subject{Kind: "ServiceAccount", ID: "service:test"},
		TTL:       time.Hour,
	}

	_, err := svc.Mint(context.Background(), req)
	if err != nil {
		t.Fatalf("mint failed: %v", err)
	}

	// Revoke it
	err = svc.Revoke(context.Background(), "default", "test-token")
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	// Verify it's marked as revoked
	resource, err := store.Get(context.Background(), KindToken, "default", "test-token")
	if err != nil {
		t.Fatalf("failed to get token after revoke: %v", err)
	}

	token, err := TokenFromResource(resource)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if !token.Status.Revoked {
		t.Error("expected token to be marked as revoked")
	}
}

func TestTokenService_MintWithMaxLifetime(t *testing.T) {
	store := newMockTokenStore()
	svc := NewTokenService(store)

	req := MintRequest{
		Name:        "test-token",
		Namespace:   "default",
		Subject:     Subject{Kind: "ServiceAccount", ID: "service:test"},
		TTL:         2 * time.Hour,
		MaxLifetime: time.Hour, // TTL exceeds max lifetime
	}

	_, err := svc.Mint(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when TTL exceeds max lifetime")
	}
}
