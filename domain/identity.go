package domain

// Subject is an authenticated caller resolved from a bearer credential by an
// IdentityVerifier. It is what the authorizer decides against.
type Subject struct {
	Kind     string   `json:"kind"`               // User | Group | EmailDomain | Token | ServiceAccount | Anonymous
	ID       string   `json:"id"`                 // provider-scoped identifier
	Email    string   `json:"email,omitempty"`    // for User subjects
	Groups   []string `json:"groups,omitempty"`   // provider-scoped groups, e.g. okta:observability
	Provider string   `json:"provider,omitempty"` // identity provider name
}

// Anonymous is the subject used when no credential is presented and the
// deployment permits it (single-user / local development).
func Anonymous() Subject {
	return Subject{Kind: "Anonymous", ID: "anonymous"}
}
