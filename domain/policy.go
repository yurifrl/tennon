package domain

import "encoding/json"

// Policy defines authorization rules for subjects accessing tools and resources.
// Compiled to Cedar in the full implementation; Phase 3 implements equivalent
// resolution semantics in Go.
type Policy struct {
	Meta ObjectMeta  `json:"metadata"`
	Spec PolicySpec  `json:"spec"`
}

// PolicySpec is the desired state of a Policy.
type PolicySpec struct {
	Rules []Rule `json:"rules"`
}

// Rule is a single authorization grant: subject × resource × effect.
type Rule struct {
	Subjects []SubjectMatch  `json:"subjects,omitempty"`
	Resource ResourceMatch   `json:"resource"`
	Effect   Effect          `json:"effect"`
}

// SubjectMatch defines what subject(s) a rule applies to.
type SubjectMatch struct {
	Kind     string `json:"kind"`               // User | Group | EmailDomain | Token | ServiceAccount
	Provider string `json:"provider,omitempty"` // identity provider name (for Group)
	Match    string `json:"match"`              // identifier or pattern
}

// ResourceMatch defines what resource(s) a rule applies to.
type ResourceMatch struct {
	Connector string `json:"connector,omitempty"` // connector name
	Tool      string `json:"tool,omitempty"`      // tool glob pattern
	Tag       string `json:"tag,omitempty"`       // connector tag
}

// ToResource marshals a Policy into a store Resource.
func (p *Policy) ToResource() (*Resource, error) {
	spec, err := json.Marshal(p.Spec)
	if err != nil {
		return nil, err
	}
	return &Resource{Kind: KindPolicy, Meta: p.Meta, Spec: spec}, nil
}

// PolicyFromResource unmarshals a store Resource into a Policy.
func PolicyFromResource(r *Resource) (*Policy, error) {
	p := &Policy{Meta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &p.Spec); err != nil {
			return nil, err
		}
	}
	return p, nil
}
