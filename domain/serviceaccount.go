package domain

import (
	"encoding/json"
	"time"
)

// ServiceAccount represents an org-owned identity with tokenless trust rules.
type ServiceAccount struct {
	ObjectMeta `json:"metadata"`
	Spec       ServiceAccountSpec   `json:"spec"`
	Status     ServiceAccountStatus `json:"status,omitempty"`
}

// ServiceAccountSpec defines the desired state.
type ServiceAccountSpec struct {
	// Trust defines how this service account authenticates (tokenless).
	Trust TrustConfig `json:"trust"`

	// Credential settings for the short-lived credentials this SA receives.
	Credential CredentialConfig `json:"credential"`

	// Subject is the resolved identity this service account represents.
	Subject Subject `json:"subject"`
}

// TrustConfig specifies the tokenless trust relationship.
type TrustConfig struct {
	// Type is the trust mechanism: oidc, kubernetes, spiffe, aws, gcp, azure.
	Type string `json:"type"`

	// Issuer is the OIDC issuer URL or trust domain.
	Issuer string `json:"issuer"`

	// Audience is the expected audience claim (for OIDC/JWT).
	Audience string `json:"audience,omitempty"`

	// Subject is the expected subject claim or pattern.
	Subject string `json:"subject,omitempty"`
}

// CredentialConfig defines short-lived credential parameters.
type CredentialConfig struct {
	// TTL is the lifetime of issued credentials (default ~1h).
	TTL time.Duration `json:"ttl,omitempty"`

	// Renewable indicates if credentials can be renewed (default false for tokenless).
	Renewable bool `json:"renewable,omitempty"`
}

// ServiceAccountStatus reflects the observed state.
type ServiceAccountStatus struct {
	// Ready indicates if the service account is properly configured.
	Ready bool `json:"ready,omitempty"`

	// LastUsed is when this service account last authenticated.
	LastUsed time.Time `json:"lastUsed,omitempty"`
}

// ToResource converts a ServiceAccount to a generic Resource.
func (sa *ServiceAccount) ToResource() (*Resource, error) {
	spec, err := json.Marshal(sa.Spec)
	if err != nil {
		return nil, err
	}
	status, err := json.Marshal(sa.Status)
	if err != nil {
		return nil, err
	}
	return &Resource{Kind: KindServiceAccount, Meta: sa.ObjectMeta, Spec: spec, Status: status}, nil
}

// ServiceAccountFromResource converts a generic Resource back to a ServiceAccount.
func ServiceAccountFromResource(r *Resource) (*ServiceAccount, error) {
	sa := &ServiceAccount{ObjectMeta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &sa.Spec); err != nil {
			return nil, err
		}
	}
	if len(r.Status) > 0 {
		if err := json.Unmarshal(r.Status, &sa.Status); err != nil {
			return nil, err
		}
	}
	return sa, nil
}
