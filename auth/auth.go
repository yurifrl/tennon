// Package auth provides identity verifiers and authorizers behind the domain
// ports. Phase 1 ships permissive defaults so the proxy path is complete;
// Phases 2-4 replace them with OIDC, tokenless trust, and Cedar-compiled Policy
// without changing any caller.
package auth

import (
	"context"

	"github.com/yurifrl/tennon/domain"
)

// AnonymousVerifier resolves any request (with or without a bearer credential)
// to a single local subject. It is the default for solo/local deployments that
// have not yet configured an identity provider.
type AnonymousVerifier struct{}

func NewAnonymousVerifier() *AnonymousVerifier { return &AnonymousVerifier{} }

func (AnonymousVerifier) Verify(_ context.Context, bearer string) (domain.Subject, error) {
	if bearer == "" {
		return domain.Anonymous(), nil
	}
	// A presented token is trusted as a local user until Phase 2 wires real
	// verification; the subject id is opaque so audit still records who called.
	return domain.Subject{Kind: "User", ID: "token:" + shortHash(bearer)}, nil
}

// AllowAllAuthorizer permits every tool call. It is the Phase 1 default; Phase 3
// replaces it with a Cedar-backed authorizer compiled from Policy objects.
type AllowAllAuthorizer struct{}

func NewAllowAllAuthorizer() *AllowAllAuthorizer { return &AllowAllAuthorizer{} }

func (AllowAllAuthorizer) Decide(context.Context, domain.Subject, domain.ToolCall) domain.Decision {
	return domain.Decision{Effect: domain.EffectAllow, Reason: "phase-1 allow-all default"}
}

// shortHash returns a short, non-reversible tag for a secret, used only for
// subject identification in logs/audit — never the secret itself.
func shortHash(s string) string {
	const hexdigits = "0123456789abcdef"
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = hexdigits[(h>>(uint(i)*4))&0xf]
	}
	return string(out)
}
