package domain

import (
	"encoding/json"
	"time"
)

// Token represents a fine-grained service token (minted, stored hashed).
type Token struct {
	ObjectMeta `json:"metadata"`
	Spec       TokenSpec   `json:"spec"`
	Status     TokenStatus `json:"status,omitempty"`
}

// TokenSpec defines the desired state of a service token.
type TokenSpec struct {
	// Subject is the identity this token represents (e.g., "service:ci-bot").
	Subject Subject `json:"subject"`
	// SecretHash is the SHA-256 hash of the token secret (stored, never the plaintext).
	SecretHash string `json:"secretHash"`
	// ExpiresAt is when this token expires.
	ExpiresAt time.Time `json:"expiresAt"`
}

// TokenStatus reflects the observed state.
type TokenStatus struct {
	// Revoked indicates if this token has been manually revoked.
	Revoked bool `json:"revoked,omitempty"`
}

// ToResource converts a Token to a generic Resource.
func (t *Token) ToResource() (*Resource, error) {
	spec, err := json.Marshal(t.Spec)
	if err != nil {
		return nil, err
	}
	status, err := json.Marshal(t.Status)
	if err != nil {
		return nil, err
	}
	return &Resource{Kind: "Token", Meta: t.ObjectMeta, Spec: spec, Status: status}, nil
}

// TokenFromResource converts a generic Resource back to a Token.
func TokenFromResource(r *Resource) (*Token, error) {
	t := &Token{ObjectMeta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &t.Spec); err != nil {
			return nil, err
		}
	}
	if len(r.Status) > 0 {
		if err := json.Unmarshal(r.Status, &t.Status); err != nil {
			return nil, err
		}
	}
	return t, nil
}
