package domain

import (
	"context"
	"encoding/json"
)

// Router is the port the controller uses to drive the data plane. The proxy
// implements it; the controller depends only on this interface, so control
// flows "up" without an upward import (see docs/arch/03-project-structure.md).
type Router interface {
	// Register makes a connector reachable and returns its exposed tools.
	Register(ctx context.Context, c *Connector) ([]Tool, error)
	// Deregister removes a connector's route and tears down its upstream.
	Deregister(ctx context.Context, c *Connector)
}

// SecretSpec is the on-store shape of a Secret's data. Values are rendered into
// upstream calls at request time and never returned by the API.
type SecretSpec struct {
	Data map[string]string `json:"data"`
}

// NewSecretResource builds a store Resource for a Secret from literal values.
func NewSecretResource(namespace, name string, data map[string]string) (*Resource, error) {
	spec, err := json.Marshal(SecretSpec{Data: data})
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = "tennon"
	}
	return &Resource{
		Kind: KindSecret,
		Meta: ObjectMeta{Name: name, Namespace: namespace},
		Spec: spec,
	}, nil
}

// SecretData decodes a Secret Resource's data map.
func SecretData(r *Resource) (map[string]string, error) {
	var spec SecretSpec
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &spec); err != nil {
			return nil, err
		}
	}
	return spec.Data, nil
}
