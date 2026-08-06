// Package controller reconciles connector objects from the store into runtime
// state. It watches the store, drives the data plane through domain.Router, and
// writes observed state back to each connector's status. It depends only on
// domain ports, never on the proxy package directly.
package controller

import (
	"context"
	"log/slog"

	"github.com/yurifrl/tennon/domain"
)

// Manager watches connectors and reconciles them into the router.
type Manager struct {
	store  domain.Store
	router domain.Router
	log    *slog.Logger
}

// NewManager builds a controller Manager.
func NewManager(store domain.Store, router domain.Router, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{store: store, router: router, log: log}
}

// connectorKinds are the connector types reconciled in Phase 1.
var connectorKinds = []string{domain.KindTennonConnector}

// Run performs an initial reconcile of existing connectors, then watches for
// changes until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) error {
	for _, kind := range connectorKinds {
		existing, err := m.store.List(ctx, kind, "")
		if err != nil {
			return err
		}
		for _, r := range existing {
			m.reconcile(ctx, r)
		}
		events, _, err := m.store.Watch(ctx, kind)
		if err != nil {
			return err
		}
		go m.watch(ctx, events)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (m *Manager) watch(ctx context.Context, events <-chan domain.WatchEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			switch ev.Type {
			case domain.WatchDeleted:
				if c, err := domain.ConnectorFromResource(ev.Resource); err == nil {
					m.router.Deregister(ctx, c)
				}
			default:
				m.reconcile(ctx, ev.Resource)
			}
		}
	}
}

// reconcile drives one connector to its desired state and records status.
func (m *Manager) reconcile(ctx context.Context, r *domain.Resource) {
	c, err := domain.ConnectorFromResource(r)
	if err != nil {
		m.log.Error("decode connector", "name", r.Meta.Name, "err", err)
		return
	}
	tools, err := m.router.Register(ctx, c)
	if err != nil {
		m.log.Warn("reconcile failed", "connector", c.Meta.Name, "err", err)
		m.setStatus(ctx, c, domain.ConnectorStatus{
			Phase:   domain.PhaseFailed,
			URL:     "/mcp/" + c.Meta.Name,
			Message: err.Error(),
		})
		return
	}
	m.setStatus(ctx, c, domain.ConnectorStatus{
		Phase:           domain.PhaseReady,
		DiscoveredTools: len(tools),
		URL:             "/mcp/" + c.Meta.Name,
	})
	m.log.Info("connector ready", "connector", c.Meta.Name, "tools", len(tools))
}

func (m *Manager) setStatus(ctx context.Context, c *domain.Connector, st domain.ConnectorStatus) {
	c.Status = st
	res, err := c.ToResource()
	if err != nil {
		m.log.Error("encode status", "connector", c.Meta.Name, "err", err)
		return
	}
	if err := m.store.UpdateStatus(ctx, res); err != nil {
		m.log.Error("write status", "connector", c.Meta.Name, "err", err)
	}
}
