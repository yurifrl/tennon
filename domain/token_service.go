package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// TokenService handles token mint and revoke operations.
type TokenService struct {
	store Store
}

// NewTokenService creates a new token service.
func NewTokenService(store Store) *TokenService {
	return &TokenService{store: store}
}

// MintRequest specifies parameters for minting a new token.
type MintRequest struct {
	Name        string
	Namespace   string
	Subject     Subject
	Grants      []TokenGrant
	TTL         time.Duration
	MaxLifetime time.Duration
}

// MintResponse contains the newly minted token (shown only once).
type MintResponse struct {
	Token     *Token
	SecretKey string // The actual token secret (tn_...), shown only once
}

// Mint creates a new token and returns the secret once.
func (s *TokenService) Mint(ctx context.Context, req MintRequest) (*MintResponse, error) {
	// Validate request
	if req.Name == "" {
		return nil, errors.New("token name is required")
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if req.TTL <= 0 {
		req.TTL = time.Hour // default 1 hour
	}
	if req.MaxLifetime > 0 && req.TTL > req.MaxLifetime {
		return nil, fmt.Errorf("TTL (%v) exceeds max lifetime (%v)", req.TTL, req.MaxLifetime)
	}

	// Generate a secure random token
	secretKey, err := generateTokenSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token secret: %w", err)
	}

	// Hash the token for storage
	hash := sha256.Sum256([]byte(secretKey))
	hashStr := hex.EncodeToString(hash[:])

	// Create the token resource
	token := &Token{
		ObjectMeta: ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Created:   time.Now(),
			Updated:   time.Now(),
		},
		Spec: TokenSpec{
			Subject:     req.Subject,
			SecretHash:  hashStr,
			ExpiresAt:   time.Now().Add(req.TTL),
			Grants:      req.Grants,
			MaxLifetime: req.MaxLifetime,
		},
		Status: TokenStatus{
			Revoked: false,
		},
	}

	// Store the token
	resource, err := token.ToResource()
	if err != nil {
		return nil, fmt.Errorf("failed to convert token to resource: %w", err)
	}

	stored, err := s.store.Apply(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	// Parse the stored token back
	storedToken, err := TokenFromResource(stored)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stored token: %w", err)
	}

	return &MintResponse{
		Token:     storedToken,
		SecretKey: secretKey,
	}, nil
}

// Revoke marks a token as revoked, immediately invalidating all credentials derived from it.
func (s *TokenService) Revoke(ctx context.Context, namespace, name string) error {
	if name == "" {
		return errors.New("token name is required")
	}
	if namespace == "" {
		namespace = "default"
	}

	// Get the token
	resource, err := s.store.Get(ctx, KindToken, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	token, err := TokenFromResource(resource)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// Mark as revoked
	token.Status.Revoked = true
	token.ObjectMeta.Updated = time.Now()

	// Update the token
	updatedResource, err := token.ToResource()
	if err != nil {
		return fmt.Errorf("failed to convert token to resource: %w", err)
	}

	if err := s.store.UpdateStatus(ctx, updatedResource); err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	return nil
}

// generateTokenSecret generates a secure random token with tn_ prefix.
func generateTokenSecret() (string, error) {
	// Generate 32 bytes of random data
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Base64 encode and add prefix
	encoded := base64.RawURLEncoding.EncodeToString(b)
	return "tn_" + encoded, nil
}
