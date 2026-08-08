package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/yurifrl/tennon/domain"
)

// TokenlessVerifier validates platform-signed tokens against ServiceAccount trust rules
// and mints short-lived credentials. Supports OIDC/Kubernetes JWTs; stubs for SPIFFE/cloud.
type TokenlessVerifier struct {
	store domain.Store
}

// NewTokenlessVerifier creates a verifier for tokenless trust.
func NewTokenlessVerifier(store domain.Store) *TokenlessVerifier {
	return &TokenlessVerifier{store: store}
}

// Verify validates a platform token against registered ServiceAccount trust rules.
func (v *TokenlessVerifier) Verify(ctx context.Context, bearer string) (domain.Subject, error) {
	if bearer == "" {
		return domain.Subject{}, errors.New("bearer token is required")
	}

	// List all service accounts
	resources, err := v.store.List(ctx, domain.KindServiceAccount, "")
	if err != nil {
		return domain.Subject{}, fmt.Errorf("failed to list service accounts: %w", err)
	}

	// Try each service account's trust rules
	for _, r := range resources {
		sa, err := domain.ServiceAccountFromResource(r)
		if err != nil {
			continue
		}

		subject, err := v.verifyAgainstTrust(ctx, bearer, sa)
		if err == nil {
			return subject, nil
		}
	}

	return domain.Subject{}, errors.New("no service account trust rule matched")
}

// verifyAgainstTrust validates a token against a specific ServiceAccount's trust config.
func (v *TokenlessVerifier) verifyAgainstTrust(ctx context.Context, bearer string, sa *domain.ServiceAccount) (domain.Subject, error) {
	switch sa.Spec.Trust.Type {
	case "oidc", "kubernetes":
		return v.verifyOIDC(ctx, bearer, sa)
	case "spiffe":
		return domain.Subject{}, errors.New("trust type spiffe not yet supported")
	case "aws":
		return domain.Subject{}, errors.New("trust type aws not yet supported")
	case "gcp":
		return domain.Subject{}, errors.New("trust type gcp not yet supported")
	case "azure":
		return domain.Subject{}, errors.New("trust type azure not yet supported")
	default:
		return domain.Subject{}, fmt.Errorf("unknown trust type: %s", sa.Spec.Trust.Type)
	}
}

// verifyOIDC validates an OIDC/Kubernetes JWT against the trust issuer's JWKS.
func (v *TokenlessVerifier) verifyOIDC(ctx context.Context, bearer string, sa *domain.ServiceAccount) (domain.Subject, error) {
	// Create a provider for the trust issuer
	provider, err := oidc.NewProvider(ctx, sa.Spec.Trust.Issuer)
	if err != nil {
		return domain.Subject{}, fmt.Errorf("failed to create OIDC provider for issuer %q: %w", sa.Spec.Trust.Issuer, err)
	}

	// Create a verifier with the expected audience
	config := &oidc.Config{}
	if sa.Spec.Trust.Audience != "" {
		config.ClientID = sa.Spec.Trust.Audience
	} else {
		config.SkipClientIDCheck = true
	}

	verifier := provider.Verifier(config)

	// Verify the token
	idToken, err := verifier.Verify(ctx, bearer)
	if err != nil {
		return domain.Subject{}, fmt.Errorf("token verification failed: %w", err)
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return domain.Subject{}, fmt.Errorf("failed to extract claims: %w", err)
	}

	// Check issuer matches
	if claims["iss"] != sa.Spec.Trust.Issuer {
		return domain.Subject{}, fmt.Errorf("issuer mismatch: expected %q, got %q", sa.Spec.Trust.Issuer, claims["iss"])
	}

	// Check audience if specified
	if sa.Spec.Trust.Audience != "" {
		aud, ok := claims["aud"].(string)
		if !ok {
			// Try array format
			audArray, ok := claims["aud"].([]interface{})
			if !ok || len(audArray) == 0 {
				return domain.Subject{}, errors.New("audience claim not found or invalid")
			}
			aud, ok = audArray[0].(string)
			if !ok {
				return domain.Subject{}, errors.New("audience claim not a string")
			}
		}
		if aud != sa.Spec.Trust.Audience {
			return domain.Subject{}, fmt.Errorf("audience mismatch: expected %q, got %q", sa.Spec.Trust.Audience, aud)
		}
	}

	// Check subject if specified
	if sa.Spec.Trust.Subject != "" {
		sub, ok := claims["sub"].(string)
		if !ok {
			return domain.Subject{}, errors.New("subject claim not found")
		}
		if sub != sa.Spec.Trust.Subject {
			return domain.Subject{}, fmt.Errorf("subject mismatch: expected %q, got %q", sa.Spec.Trust.Subject, sub)
		}
	}

	// All checks passed - return the service account's configured subject
	// with a short-lived credential TTL
	subject := sa.Spec.Subject
	subject.Provider = "tokenless:" + sa.Spec.Trust.Type
	
	// Note: In a full implementation, we would store credential expiry in the subject
	// or return it separately. For now, the subject is returned and the caller
	// should enforce the TTL (default ~1h as documented).
	return subject, nil
}
