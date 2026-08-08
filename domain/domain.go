// Package domain holds Tennon's core types and the port interfaces the outer
// layers must satisfy. It imports nothing from Kubernetes, the store, the HTTP
// stack, the proxy, or any vendor SDK — only the Go standard library.
//
// The dependency rule (docs/arch/03-project-structure.md): packages may depend
// only on packages below them. When a lower layer needs behaviour that lives
// higher up, it declares an interface here and the higher layer implements it.
package domain

import "time"

// ObjectMeta is the metadata shared by every Tennon resource, mirroring the
// Kubernetes object metadata that the CRDs expose.
type ObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Generation  int64             `json:"generation,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
	Updated     time.Time         `json:"updated,omitempty"`
}

// APIGroup and version for every Tennon custom resource.
const (
	APIGroup   = "tennon.io"
	APIVersion = "v1alpha1"
)

// Resource kinds.
const (
	KindTennonConnector  = "TennonConnector"
	KindOpenApiConnector = "OpenApiConnector"
	KindGraphqlConnector = "GraphqlConnector"
	KindOAuthApp         = "OAuthApp"
	KindPolicy           = "Policy"
	KindToken            = "Token"
	KindServiceAccount   = "ServiceAccount"
	KindSecret           = "Secret"
	KindExecution        = "Execution"
)
