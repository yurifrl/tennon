package auth

import (
	"context"
	"testing"

	"github.com/yurifrl/tennon/domain"
)

type mockTokenlessStore struct {
	serviceAccounts []*domain.Resource
}

func (m *mockTokenlessStore) List(_ context.Context, kind, _ string) ([]*domain.Resource, error) {
	if kind == domain.KindServiceAccount {
		return m.serviceAccounts, nil
	}
	return nil, nil
}

func (m *mockTokenlessStore) Get(_ context.Context, _, _, _ string) (*domain.Resource, error) {
	return nil, nil
}

func (m *mockTokenlessStore) Apply(_ context.Context, _ *domain.Resource) (*domain.Resource, error) {
	return nil, nil
}

func (m *mockTokenlessStore) UpdateStatus(_ context.Context, _ *domain.Resource) error {
	return nil
}

func (m *mockTokenlessStore) Delete(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockTokenlessStore) Watch(_ context.Context, _ string) (<-chan domain.WatchEvent, func(), error) {
	return nil, nil, nil
}

func (m *mockTokenlessStore) Close() error {
	return nil
}

func TestTokenlessVerifier_OIDC(t *testing.T) {
	// TODO: Full OIDC test requires complex mock server setup with proper discovery
	// For now, verify the basic structure works
	t.Skip("OIDC verification test requires full mock OIDC server setup")
}

func TestTokenlessVerifier_UnsupportedTrustType(t *testing.T) {
	sa := &domain.ServiceAccount{
		ObjectMeta: domain.ObjectMeta{Name: "test-sa"},
		Spec: domain.ServiceAccountSpec{
			Trust: domain.TrustConfig{Type: "spiffe"},
			Subject: domain.Subject{Kind: "ServiceAccount", ID: "service:test-sa"},
		},
	}

	saResource, _ := sa.ToResource()
	store := &mockTokenlessStore{serviceAccounts: []*domain.Resource{saResource}}
	verifier := NewTokenlessVerifier(store)

	_, err := verifier.Verify(context.Background(), "any-token")
	if err == nil {
		t.Fatal("expected error for unsupported trust type")
	}
	// The error should indicate either no match or unsupported type
	if err.Error() != "trust type spiffe not yet supported" && err.Error() != "no service account trust rule matched" {
		t.Errorf("unexpected error: %v", err)
	}
}
