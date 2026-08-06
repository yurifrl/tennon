// Package secret implements domain.SecretRenderer: it resolves
// {{secret:name.key}} templates against Secret objects in the store and renders
// them into credential headers/env at call time. Rendered values are returned
// only to the proxy/controller and never leave the process.
package secret

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/yurifrl/tennon/domain"
)

var tmplRe = regexp.MustCompile(`\{\{\s*secret:([a-zA-Z0-9._-]+)\.([a-zA-Z0-9._-]+)\s*\}\}`)

// Renderer resolves secret templates from a domain.Store.
type Renderer struct {
	store     domain.Store
	namespace string // default namespace for Secret lookups
}

// NewRenderer returns a Renderer backed by store. defaultNamespace is used when
// a connector carries no namespace.
func NewRenderer(store domain.Store, defaultNamespace string) *Renderer {
	if defaultNamespace == "" {
		defaultNamespace = "tennon"
	}
	return &Renderer{store: store, namespace: defaultNamespace}
}

// RenderString replaces every {{secret:name.key}} in template with its value.
func (r *Renderer) RenderString(ctx context.Context, namespace, template string) (string, error) {
	if namespace == "" {
		namespace = r.namespace
	}
	if !strings.Contains(template, "{{") {
		return template, nil
	}
	cache := map[string]map[string]string{}
	var renderErr error
	out := tmplRe.ReplaceAllStringFunc(template, func(m string) string {
		sub := tmplRe.FindStringSubmatch(m)
		name, key := sub[1], sub[2]
		data, ok := cache[name]
		if !ok {
			d, err := r.loadSecret(ctx, namespace, name)
			if err != nil {
				renderErr = err
				return m
			}
			data = d
			cache[name] = d
		}
		val, ok := data[key]
		if !ok {
			renderErr = fmt.Errorf("secret %q has no key %q", name, key)
			return m
		}
		return val
	})
	if renderErr != nil {
		return "", renderErr
	}
	return out, nil
}

// Render renders every value in templates, returning a new map.
func (r *Renderer) Render(ctx context.Context, namespace string, templates map[string]string) (map[string]string, error) {
	if len(templates) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(templates))
	for k, v := range templates {
		rendered, err := r.RenderString(ctx, namespace, v)
		if err != nil {
			return nil, err
		}
		out[k] = rendered
	}
	return out, nil
}

func (r *Renderer) loadSecret(ctx context.Context, namespace, name string) (map[string]string, error) {
	res, err := r.store.Get(ctx, domain.KindSecret, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("resolve secret %q: %w", name, err)
	}
	data, err := domain.SecretData(res)
	if err != nil {
		return nil, fmt.Errorf("decode secret %q: %w", name, err)
	}
	return data, nil
}
