// Package store provides domain.Store implementations. All drivers share the
// same CRD types; they differ only in where objects are persisted.
package store

import (
	"context"
	"sync"

	"github.com/yurifrl/tennon/domain"
)

// Memory is an in-memory domain.Store with get/list/watch semantics. It is the
// default for tests and ephemeral single-process runs.
type Memory struct {
	mu      sync.RWMutex
	objects map[string]*domain.Resource // key: kind/namespace/name
	watches map[string]map[int]chan domain.WatchEvent
	nextID  int
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		objects: map[string]*domain.Resource{},
		watches: map[string]map[int]chan domain.WatchEvent{},
	}
}

func key(kind, ns, name string) string { return kind + "/" + ns + "/" + name }

func clone(r *domain.Resource) *domain.Resource {
	if r == nil {
		return nil
	}
	cp := *r
	if r.Meta.Labels != nil {
		cp.Meta.Labels = map[string]string{}
		for k, v := range r.Meta.Labels {
			cp.Meta.Labels[k] = v
		}
	}
	if len(r.Spec) > 0 {
		cp.Spec = append([]byte(nil), r.Spec...)
	}
	if len(r.Status) > 0 {
		cp.Status = append([]byte(nil), r.Status...)
	}
	return &cp
}

func (m *Memory) Get(_ context.Context, kind, ns, name string) (*domain.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.objects[key(kind, ns, name)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return clone(r), nil
}

func (m *Memory) List(_ context.Context, kind, ns string) ([]*domain.Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*domain.Resource
	for _, r := range m.objects {
		if r.Kind != kind {
			continue
		}
		if ns != "" && r.Meta.Namespace != ns {
			continue
		}
		out = append(out, clone(r))
	}
	return out, nil
}

func (m *Memory) Apply(_ context.Context, r *domain.Resource) (*domain.Resource, error) {
	m.mu.Lock()
	k := key(r.Kind, r.Meta.Namespace, r.Meta.Name)
	existing, found := m.objects[k]
	stored := clone(r)
	evType := domain.WatchAdded
	if found {
		stored.Status = existing.Status // Apply never clobbers status
		stored.Meta.Created = existing.Meta.Created
		stored.Meta.Generation = existing.Meta.Generation + 1
		evType = domain.WatchModified
	}
	m.objects[k] = stored
	m.mu.Unlock()
	m.broadcast(stored.Kind, domain.WatchEvent{Type: evType, Resource: clone(stored)})
	return clone(stored), nil
}

func (m *Memory) UpdateStatus(_ context.Context, r *domain.Resource) error {
	m.mu.Lock()
	k := key(r.Kind, r.Meta.Namespace, r.Meta.Name)
	existing, ok := m.objects[k]
	if !ok {
		m.mu.Unlock()
		return domain.ErrNotFound
	}
	existing.Status = append([]byte(nil), r.Status...)
	updated := clone(existing)
	m.mu.Unlock()
	m.broadcast(r.Kind, domain.WatchEvent{Type: domain.WatchModified, Resource: updated})
	return nil
}

func (m *Memory) Delete(_ context.Context, kind, ns, name string) error {
	m.mu.Lock()
	k := key(kind, ns, name)
	r, ok := m.objects[k]
	if !ok {
		m.mu.Unlock()
		return domain.ErrNotFound
	}
	delete(m.objects, k)
	deleted := clone(r)
	m.mu.Unlock()
	m.broadcast(kind, domain.WatchEvent{Type: domain.WatchDeleted, Resource: deleted})
	return nil
}

func (m *Memory) Watch(ctx context.Context, kind string) (<-chan domain.WatchEvent, func(), error) {
	m.mu.Lock()
	if m.watches[kind] == nil {
		m.watches[kind] = map[int]chan domain.WatchEvent{}
	}
	id := m.nextID
	m.nextID++
	ch := make(chan domain.WatchEvent, 64)
	m.watches[kind][id] = ch
	m.mu.Unlock()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			m.mu.Lock()
			if subs := m.watches[kind]; subs != nil {
				delete(subs, id)
			}
			m.mu.Unlock()
			close(ch)
		})
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ch, stop, nil
}

func (m *Memory) broadcast(kind string, ev domain.WatchEvent) {
	m.mu.RLock()
	subs := make([]chan domain.WatchEvent, 0, len(m.watches[kind]))
	for _, ch := range m.watches[kind] {
		subs = append(subs, ch)
	}
	m.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default: // drop if a slow watcher's buffer is full; reconcile is level-based
		}
	}
}

func (m *Memory) Close() error { return nil }
