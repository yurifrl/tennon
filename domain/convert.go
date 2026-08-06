package domain

import (
	"encoding/json"
	"path"
	"strings"
)

// ToResource marshals a Connector into a store Resource.
func (c *Connector) ToResource() (*Resource, error) {
	spec, err := json.Marshal(c.Spec)
	if err != nil {
		return nil, err
	}
	status, err := json.Marshal(c.Status)
	if err != nil {
		return nil, err
	}
	kind := c.Kind
	if kind == "" {
		kind = KindTennonConnector
	}
	return &Resource{Kind: kind, Meta: c.Meta, Spec: spec, Status: status}, nil
}

// ConnectorFromResource unmarshals a store Resource into a Connector.
func ConnectorFromResource(r *Resource) (*Connector, error) {
	c := &Connector{Kind: r.Kind, Meta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &c.Spec); err != nil {
			return nil, err
		}
	}
	if len(r.Status) > 0 {
		if err := json.Unmarshal(r.Status, &c.Status); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// ExposedName applies the connector's exposePrefix to an upstream tool name.
func (s ToolsSpec) ExposedName(upstream string) string {
	if s.ExposePrefix == "" {
		return upstream
	}
	if strings.HasPrefix(upstream, s.ExposePrefix) {
		return upstream
	}
	return s.ExposePrefix + upstream
}

// UpstreamName strips the exposePrefix from a caller-facing tool name.
func (s ToolsSpec) UpstreamName(exposed string) string {
	if s.ExposePrefix != "" && strings.HasPrefix(exposed, s.ExposePrefix) {
		return strings.TrimPrefix(exposed, s.ExposePrefix)
	}
	return exposed
}

// Permits reports whether an upstream tool name passes the allow/deny globs.
// Deny wins over allow; an empty allow list allows everything not denied.
func (s ToolsSpec) Permits(upstream string) bool {
	for _, g := range s.Deny {
		if matchGlob(g, upstream) {
			return false
		}
	}
	if len(s.Allow) == 0 {
		return true
	}
	for _, g := range s.Allow {
		if matchGlob(g, upstream) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, name string) bool {
	if pattern == "*" || pattern == name {
		return true
	}
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}
