package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/yurifrl/tennon/domain"
)

// OIDCVerifier validates OIDC access/JWT tokens against an issuer's JWKS.
type OIDCVerifier struct {
	provider       *oidc.Provider
	verifier       *oidc.IDTokenVerifier
	clientID       string
	usernameClaim  string
	groupsClaim    string
	providerName   string
}

// OIDCConfig holds configuration for OIDC verification.
type OIDCConfig struct {
	IssuerURL     string
	ClientID      string
	UsernameClaim string // defaults to "email"
	GroupsClaim   string // defaults to "groups"
	ProviderName  string // e.g., "google", "okta"
}

// NewOIDCVerifier creates an OIDC verifier that validates tokens against the issuer's JWKS.
func NewOIDCVerifier(ctx context.Context, cfg OIDCConfig) (*OIDCVerifier, error) {
	if cfg.IssuerURL == "" {
		return nil, errors.New("issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	usernameClaim := cfg.UsernameClaim
	if usernameClaim == "" {
		usernameClaim = "email"
	}

	groupsClaim := cfg.GroupsClaim
	if groupsClaim == "" {
		groupsClaim = "groups"
	}

	return &OIDCVerifier{
		provider:      provider,
		verifier:      verifier,
		clientID:      cfg.ClientID,
		usernameClaim: usernameClaim,
		groupsClaim:   groupsClaim,
		providerName:  cfg.ProviderName,
	}, nil
}

// Verify validates an OIDC token and extracts subject, email, and groups.
func (v *OIDCVerifier) Verify(ctx context.Context, bearer string) (domain.Subject, error) {
	if bearer == "" {
		return domain.Subject{}, errors.New("bearer token is required")
	}

	// Verify the token signature, issuer, audience, and expiry
	idToken, err := v.verifier.Verify(ctx, bearer)
	if err != nil {
		return domain.Subject{}, fmt.Errorf("token verification failed: %w", err)
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return domain.Subject{}, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Extract subject (username)
	var id string
	if username, ok := claims[v.usernameClaim].(string); ok {
		id = username
	} else {
		// Fallback to "sub" claim if username claim is not found
		if sub, ok := claims["sub"].(string); ok {
			id = sub
		} else {
			return domain.Subject{}, fmt.Errorf("username claim %q not found in token", v.usernameClaim)
		}
	}

	// Extract email
	email, _ := claims["email"].(string)

	// Extract groups
	var groups []string
	if groupsVal, ok := claims[v.groupsClaim]; ok {
		switch g := groupsVal.(type) {
		case []interface{}:
			for _, item := range g {
				if str, ok := item.(string); ok {
					groups = append(groups, str)
				}
			}
		case []string:
			groups = g
		}
	}

	return domain.Subject{
		Kind:     "User",
		ID:       id,
		Email:    email,
		Groups:   groups,
		Provider: v.providerName,
	}, nil
}

// ServiceTokenVerifier validates service tokens (tn_...) against stored hashes.
type ServiceTokenVerifier struct {
	store domain.Store
}

// NewServiceTokenVerifier creates a verifier for service tokens.
func NewServiceTokenVerifier(store domain.Store) *ServiceTokenVerifier {
	return &ServiceTokenVerifier{store: store}
}

// Verify validates a service token by hashing and looking it up in the store.
func (v *ServiceTokenVerifier) Verify(ctx context.Context, bearer string) (domain.Subject, error) {
	if bearer == "" {
		return domain.Subject{}, errors.New("bearer token is required")
	}

	// Service tokens have a tn_ prefix
	if !strings.HasPrefix(bearer, "tn_") {
		return domain.Subject{}, errors.New("invalid service token format")
	}

	// Hash the token
	hash := sha256.Sum256([]byte(bearer))
	hashStr := hex.EncodeToString(hash[:])

	// List all tokens and find a match
	resources, err := v.store.List(ctx, "Token", "tennon")
	if err != nil {
		return domain.Subject{}, fmt.Errorf("failed to list tokens: %w", err)
	}

	for _, r := range resources {
		token, err := domain.TokenFromResource(r)
		if err != nil {
			continue
		}

		// Check if hash matches
		if token.Spec.SecretHash == hashStr {
			// Check if token is expired
			if time.Now().After(token.Spec.ExpiresAt) {
				return domain.Subject{}, errors.New("token expired")
			}

			// Check if token is revoked
			if token.Status.Revoked {
				return domain.Subject{}, errors.New("token revoked")
			}

			return token.Spec.Subject, nil
		}
	}

	return domain.Subject{}, errors.New("invalid or unknown token")
}

// ChainedVerifier tries multiple verifiers in order until one succeeds.
type ChainedVerifier struct {
	verifiers []domain.IdentityVerifier
}

// NewChainedVerifier creates a verifier that tries multiple verifiers in order.
func NewChainedVerifier(verifiers ...domain.IdentityVerifier) *ChainedVerifier {
	return &ChainedVerifier{verifiers: verifiers}
}

// Verify tries each verifier in order until one succeeds.
func (c *ChainedVerifier) Verify(ctx context.Context, bearer string) (domain.Subject, error) {
	var lastErr error
	for _, v := range c.verifiers {
		subject, err := v.Verify(ctx, bearer)
		if err == nil {
			return subject, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return domain.Subject{}, lastErr
	}
	return domain.Subject{}, errors.New("no verifier succeeded")
}

// NewVerifierFromConfig creates the appropriate verifier based on configuration.
// Returns AnonymousVerifier when no OIDC issuer is configured, otherwise returns
// a chained verifier that tries service tokens first, then OIDC.
func NewVerifierFromConfig(ctx context.Context, cfg OIDCConfig, store domain.Store) (domain.IdentityVerifier, error) {
	// If no OIDC issuer is configured, return anonymous verifier
	if cfg.IssuerURL == "" {
		return NewAnonymousVerifier(), nil
	}

	// Create OIDC verifier
	oidcVerifier, err := NewOIDCVerifier(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC verifier: %w", err)
	}

	// Create service token verifier
	serviceTokenVerifier := NewServiceTokenVerifier(store)

	// Chain them: try service tokens first, then OIDC
	return NewChainedVerifier(serviceTokenVerifier, oidcVerifier), nil
}
