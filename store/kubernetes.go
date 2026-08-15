package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yurifrl/tennon/domain"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Kubernetes persists resources as CRDs in a Kubernetes cluster via
// controller-runtime. Watch notifications come from the Kubernetes API server's
// native watch mechanism.
type Kubernetes struct {
	client client.Client
	b      *broadcaster
}

// NewKubernetes creates a Kubernetes-backed store using the provided client.
func NewKubernetes(c client.Client) *Kubernetes {
	return &Kubernetes{client: c, b: newBroadcaster()}
}

func (k *Kubernetes) gvr(kind string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    domain.APIGroup,
		Version:  domain.APIVersion,
		Resource: pluralize(kind),
	}
}

func pluralize(kind string) string {
	// Simple pluralization: add 's' or 'es' based on suffix
	switch {
	case kind == "Policy":
		return "policies"
	default:
		return kind + "s"
	}
}

func (k *Kubernetes) resourceToUnstructured(r *domain.Resource) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(domain.APIGroup + "/" + domain.APIVersion)
	obj.SetKind(r.Kind)
	obj.SetName(r.Meta.Name)
	obj.SetNamespace(r.Meta.Namespace)
	obj.SetLabels(r.Meta.Labels)
	obj.SetAnnotations(r.Meta.Annotations)
	obj.SetGeneration(r.Meta.Generation)

	// Set spec
	if len(r.Spec) > 0 {
		var spec map[string]interface{}
		if err := json.Unmarshal(r.Spec, &spec); err != nil {
			return nil, fmt.Errorf("unmarshal spec: %w", err)
		}
		if err := unstructured.SetNestedField(obj.Object, spec, "spec"); err != nil {
			return nil, fmt.Errorf("set spec: %w", err)
		}
	}

	// Set status
	if len(r.Status) > 0 {
		var status map[string]interface{}
		if err := json.Unmarshal(r.Status, &status); err != nil {
			return nil, fmt.Errorf("unmarshal status: %w", err)
		}
		if err := unstructured.SetNestedField(obj.Object, status, "status"); err != nil {
			return nil, fmt.Errorf("set status: %w", err)
		}
	}

	return obj, nil
}

func (k *Kubernetes) unstructuredToResource(obj *unstructured.Unstructured) (*domain.Resource, error) {
	r := &domain.Resource{
		Kind: obj.GetKind(),
		Meta: domain.ObjectMeta{
			Name:        obj.GetName(),
			Namespace:   obj.GetNamespace(),
			Labels:      obj.GetLabels(),
			Annotations: obj.GetAnnotations(),
			Generation:  obj.GetGeneration(),
			Created:     obj.GetCreationTimestamp().Time,
		},
	}

	// Extract spec
	if spec, found, err := unstructured.NestedMap(obj.Object, "spec"); err != nil {
		return nil, fmt.Errorf("get spec: %w", err)
	} else if found {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return nil, fmt.Errorf("marshal spec: %w", err)
		}
		r.Spec = specBytes
	}

	// Extract status
	if status, found, err := unstructured.NestedMap(obj.Object, "status"); err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	} else if found {
		statusBytes, err := json.Marshal(status)
		if err != nil {
			return nil, fmt.Errorf("marshal status: %w", err)
		}
		r.Status = statusBytes
	}

	// Use last modified time as updated
	if lastModified := obj.GetManagedFields(); len(lastModified) > 0 {
		r.Meta.Updated = lastModified[len(lastModified)-1].Time.Time
	} else {
		r.Meta.Updated = r.Meta.Created
	}

	return r, nil
}

func (k *Kubernetes) Get(ctx context.Context, kind, ns, name string) (*domain.Resource, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(domain.APIGroup + "/" + domain.APIVersion)
	obj.SetKind(kind)

	key := client.ObjectKey{Namespace: ns, Name: name}
	if err := k.client.Get(ctx, key, obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return k.unstructuredToResource(obj)
}

func (k *Kubernetes) List(ctx context.Context, kind, ns string) ([]*domain.Resource, error) {
	list := &unstructured.UnstructuredList{}
	list.SetAPIVersion(domain.APIGroup + "/" + domain.APIVersion)
	list.SetKind(kind + "List")

	opts := []client.ListOption{}
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}

	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}

	resources := make([]*domain.Resource, 0, len(list.Items))
	for i := range list.Items {
		r, err := k.unstructuredToResource(&list.Items[i])
		if err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}

	return resources, nil
}

func (k *Kubernetes) Apply(ctx context.Context, r *domain.Resource) (*domain.Resource, error) {
	obj, err := k.resourceToUnstructured(r)
	if err != nil {
		return nil, err
	}

	// Check if exists
	existing, getErr := k.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	evType := domain.WatchAdded

	if getErr == nil {
		// Update existing resource
		evType = domain.WatchModified
		obj.SetResourceVersion(fmt.Sprintf("%d", existing.Meta.Generation))
		if err := k.client.Update(ctx, obj); err != nil {
			return nil, err
		}
	} else if getErr == domain.ErrNotFound {
		// Create new resource
		if err := k.client.Create(ctx, obj); err != nil {
			return nil, err
		}
	} else {
		return nil, getErr
	}

	// Get the updated resource
	stored, err := k.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return nil, err
	}

	// Broadcast the event
	k.b.broadcast(r.Kind, domain.WatchEvent{Type: evType, Resource: stored})

	return stored, nil
}

func (k *Kubernetes) UpdateStatus(ctx context.Context, r *domain.Resource) error {
	obj, err := k.resourceToUnstructured(r)
	if err != nil {
		return err
	}

	// Get current resource to preserve spec and get resource version
	current, err := k.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}

	// Set the spec from current to preserve it
	if len(current.Spec) > 0 {
		var spec map[string]interface{}
		if err := json.Unmarshal(current.Spec, &spec); err != nil {
			return fmt.Errorf("unmarshal current spec: %w", err)
		}
		if err := unstructured.SetNestedField(obj.Object, spec, "spec"); err != nil {
			return fmt.Errorf("set spec: %w", err)
		}
	}

	if err := k.client.Status().Update(ctx, obj); err != nil {
		return err
	}

	// Get the updated resource
	stored, err := k.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}

	// Broadcast the event
	k.b.broadcast(r.Kind, domain.WatchEvent{Type: domain.WatchModified, Resource: stored})

	return nil
}

func (k *Kubernetes) Delete(ctx context.Context, kind, ns, name string) error {
	existing, err := k.Get(ctx, kind, ns, name)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(domain.APIGroup + "/" + domain.APIVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace(ns)

	if err := k.client.Delete(ctx, obj); err != nil {
		return err
	}

	// Broadcast the event
	k.b.broadcast(kind, domain.WatchEvent{Type: domain.WatchDeleted, Resource: existing})

	return nil
}

func (k *Kubernetes) Watch(ctx context.Context, kind string) (<-chan domain.WatchEvent, func(), error) {
	// Subscribe to our broadcaster for local events
	ch, stop := k.b.subscribe(kind)

	go func() {
		<-ctx.Done()
		stop()
	}()

	return ch, stop, nil
}

func (k *Kubernetes) Close() error {
	// No cleanup needed for controller-runtime client
	return nil
}
