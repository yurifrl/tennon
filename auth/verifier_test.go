package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/yurifrl/tennon/domain"
)

// mockStore is a simple in-memory store for testing.
type mockStore struct {
	tokens map[string]*domain.Resource
}

func newMockStore() *mockStore {
	return &mockStore{tokens: make(map[string]*domain.Resource)}
}

func (m *mockStore) Get(ctx context.Context, kind, namespace, name string) (*domain.Resource, error) {
	key := kind + "/" + namespace + "/" + name
	if r, ok := m.tokens[key]; ok {
		return r, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockStore) List(ctx context.Context, kind, namespace string) ([]*domain.Resource, error) {
	var results []*domain.Resource
	prefix := kind + "/" + namespace + "/"
	for k, v := range m.tokens {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			results = append(results, v)
		}
	}
	return results, nil
}

func (m *mockStore) Apply(ctx context.Context, r *domain.Resource) (*domain.Resource, error) {
	key := r.Kind + "/" + r.Meta.Namespace + "/" + r.Meta.Name
	m.tokens[key] = r
	return r, nil
}

func (m *mockStore) UpdateStatus(ctx context.Context, r *domain.Resource) error {
	key := r.Kind + "/" + r.Meta.Namespace + "/" + r.Meta.Name
	if _, ok := m.tokens[key]; !ok {
		return domain.ErrNotFound
	}
	m.tokens[key].Status = r.Status
	return nil
}

func (m *mockStore) Delete(ctx context.Context, kind, namespace, name string) error {
	key := kind + "/" + namespace + "/" + name
	delete(m.tokens, key)
	return nil
}

func (m *mockStore) Watch(ctx context.Context, kind string) (<-chan domain.WatchEvent, func(), error) {
	ch := make(chan domain.WatchEvent)
	stop := func() { close(ch) }
	return ch, stop, nil
}

func (m *mockStore) Close() error { return nil }

// mockOIDCServer creates a test OIDC provider with JWKS endpoint.
func mockOIDCServer(t *testing.T, privateKey *rsa.PrivateKey) (*httptest.Server, string) {
	publicKey := &privateKey.PublicKey

	// Create JWKS
	jwk := jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     "test-key",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	server := httptest.NewServer(nil)
	issuer := server.URL

	mux := http.NewServeMux()

	// OIDC discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := map[string]interface{}{
			"issuer":   issuer,
			"jwks_uri": issuer + "/jwks",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	// JWKS endpoint
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	server.Config.Handler = mux
	return server, issuer
}

// createTestToken creates a signed JWT token for testing.
func createTestToken(t *testing.T, privateKey *rsa.PrivateKey, issuer, clientID, email string, groups []string, expiry time.Time) string {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	claims := map[string]interface{}{
		"iss":    issuer,
		"aud":    clientID,
		"sub":    "user123",
		"email":  email,
		"groups": groups,
		"exp":    expiry.Unix(),
		"iat":    time.Now().Unix(),
	}

	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	return token
}

func TestOIDCVerifier(t *testing.T) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Start mock OIDC server
	server, issuer := mockOIDCServer(t, privateKey)
	defer server.Close()
	clientID := "test-client"

	ctx := context.Background()

	// Create OIDC verifier
	verifier, err := NewOIDCVerifier(ctx, OIDCConfig{
		IssuerURL:    issuer,
		ClientID:     clientID,
		ProviderName: "test",
	})
	if err != nil {
		t.Fatalf("failed to create OIDC verifier: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		token := createTestToken(t, privateKey, issuer, clientID, "user@example.com", []string{"group1", "group2"}, time.Now().Add(time.Hour))

		subject, err := verifier.Verify(ctx, token)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if subject.Email != "user@example.com" {
			t.Errorf("expected email user@example.com, got %s", subject.Email)
		}
		if len(subject.Groups) != 2 {
			t.Errorf("expected 2 groups, got %d", len(subject.Groups))
		}
		if subject.Provider != "test" {
			t.Errorf("expected provider test, got %s", subject.Provider)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := createTestToken(t, privateKey, issuer, clientID, "user@example.com", []string{"group1"}, time.Now().Add(-time.Hour))

		_, err := verifier.Verify(ctx, token)
		if err == nil {
			t.Error("expected verification to fail for expired token")
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		token := createTestToken(t, privateKey, issuer, "wrong-client", "user@example.com", []string{"group1"}, time.Now().Add(time.Hour))

		_, err := verifier.Verify(ctx, token)
		if err == nil {
			t.Error("expected verification to fail for wrong audience")
		}
	})

	t.Run("empty bearer", func(t *testing.T) {
		_, err := verifier.Verify(ctx, "")
		if err == nil {
			t.Error("expected verification to fail for empty bearer")
		}
	})
}

func TestServiceTokenVerifier(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	verifier := NewServiceTokenVerifier(store)

	secret := "tn_test_secret_token_12345"
	hash := sha256.Sum256([]byte(secret))
	hashStr := hex.EncodeToString(hash[:])

	// Create a test token in the store
	token := &domain.Token{
		ObjectMeta: domain.ObjectMeta{
			Namespace: "tennon",
			Name:      "test-token",
		},
		Spec: domain.TokenSpec{
			Subject: domain.Subject{
				Kind:  "ServiceAccount",
				ID:    "service:ci-bot",
				Email: "ci-bot@example.com",
			},
			SecretHash: hashStr,
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}
	tokenResource, _ := token.ToResource()
	store.Apply(ctx, tokenResource)

	t.Run("valid service token", func(t *testing.T) {
		subject, err := verifier.Verify(ctx, secret)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if subject.Kind != "ServiceAccount" {
			t.Errorf("expected kind ServiceAccount, got %s", subject.Kind)
		}
		if subject.ID != "service:ci-bot" {
			t.Errorf("expected ID service:ci-bot, got %s", subject.ID)
		}
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := verifier.Verify(ctx, "invalid_token")
		if err == nil {
			t.Error("expected verification to fail for invalid format")
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		_, err := verifier.Verify(ctx, "tn_unknown_token")
		if err == nil {
			t.Error("expected verification to fail for unknown token")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expiredSecret := "tn_expired_token"
		expiredHash := sha256.Sum256([]byte(expiredSecret))
		expiredHashStr := hex.EncodeToString(expiredHash[:])

		expiredToken := &domain.Token{
			ObjectMeta: domain.ObjectMeta{
				Namespace: "tennon",
				Name:      "expired-token",
			},
			Spec: domain.TokenSpec{
				Subject: domain.Subject{
					Kind: "ServiceAccount",
					ID:   "service:expired",
				},
				SecretHash: expiredHashStr,
				ExpiresAt:  time.Now().Add(-time.Hour),
			},
		}
		expiredResource, _ := expiredToken.ToResource()
		store.Apply(ctx, expiredResource)

		_, err := verifier.Verify(ctx, expiredSecret)
		if err == nil {
			t.Error("expected verification to fail for expired token")
		}
	})

	t.Run("revoked token", func(t *testing.T) {
		revokedSecret := "tn_revoked_token"
		revokedHash := sha256.Sum256([]byte(revokedSecret))
		revokedHashStr := hex.EncodeToString(revokedHash[:])

		revokedToken := &domain.Token{
			ObjectMeta: domain.ObjectMeta{
				Namespace: "tennon",
				Name:      "revoked-token",
			},
			Spec: domain.TokenSpec{
				Subject: domain.Subject{
					Kind: "ServiceAccount",
					ID:   "service:revoked",
				},
				SecretHash: revokedHashStr,
				ExpiresAt:  time.Now().Add(24 * time.Hour),
			},
			Status: domain.TokenStatus{
				Revoked: true,
			},
		}
		revokedResource, _ := revokedToken.ToResource()
		store.Apply(ctx, revokedResource)

		_, err := verifier.Verify(ctx, revokedSecret)
		if err == nil {
			t.Error("expected verification to fail for revoked token")
		}
	})
}

func TestChainedVerifier(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	// Create a service token
	secret := "tn_test_token"
	hash := sha256.Sum256([]byte(secret))
	hashStr := hex.EncodeToString(hash[:])

	token := &domain.Token{
		ObjectMeta: domain.ObjectMeta{
			Namespace: "tennon",
			Name:      "test-token",
		},
		Spec: domain.TokenSpec{
			Subject: domain.Subject{
				Kind: "ServiceAccount",
				ID:   "service:test",
			},
			SecretHash: hashStr,
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}
	tokenResource, _ := token.ToResource()
	store.Apply(ctx, tokenResource)

	serviceVerifier := NewServiceTokenVerifier(store)
	anonymousVerifier := NewAnonymousVerifier()

	chained := NewChainedVerifier(serviceVerifier, anonymousVerifier)

	t.Run("service token succeeds", func(t *testing.T) {
		subject, err := chained.Verify(ctx, secret)
		if err != nil {
			t.Fatalf("chained verification failed: %v", err)
		}

		if subject.Kind != "ServiceAccount" {
			t.Errorf("expected ServiceAccount, got %s", subject.Kind)
		}
	})

	t.Run("fallback to anonymous", func(t *testing.T) {
		subject, err := chained.Verify(ctx, "random_bearer_token")
		// This should fail on service verifier but anonymous will handle it
		if err != nil {
			t.Fatalf("chained verification failed: %v", err)
		}

		if subject.Kind != "User" {
			t.Errorf("expected User (from anonymous), got %s", subject.Kind)
		}
	})
}

func TestNewVerifierFromConfig(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	t.Run("no OIDC config returns anonymous", func(t *testing.T) {
		verifier, err := NewVerifierFromConfig(ctx, OIDCConfig{}, store)
		if err != nil {
			t.Fatalf("failed to create verifier: %v", err)
		}

		if _, ok := verifier.(*AnonymousVerifier); !ok {
			t.Error("expected AnonymousVerifier when no OIDC config provided")
		}
	})

	t.Run("OIDC config without client ID fails", func(t *testing.T) {
		_, err := NewVerifierFromConfig(ctx, OIDCConfig{
			IssuerURL: "https://example.com",
		}, store)
		if err == nil {
			t.Error("expected error when client ID is missing")
		}
	})
}
