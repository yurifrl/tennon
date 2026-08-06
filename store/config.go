package store

import (
	"fmt"

	"github.com/yurifrl/tennon/domain"
)

// Config selects and parameterises a store backend.
type Config struct {
	Backend string // memory | sqlite | postgres | kubernetes
	Path    string // sqlite file path
	DSN     string // postgres DSN
}

// NewFromConfig builds the domain.Store selected by cfg. This is called from
// the composition root (cmd/) — the only place that knows concrete types.
func NewFromConfig(cfg Config) (domain.Store, error) {
	switch cfg.Backend {
	case "", "memory":
		return NewMemory(), nil
	case "sqlite":
		if cfg.Path == "" {
			cfg.Path = "tennon.db"
		}
		return NewSQLite(cfg.Path)
	case "postgres":
		return nil, fmt.Errorf("store: postgres backend not yet implemented (roadmap phase 2)")
	case "kubernetes":
		return nil, fmt.Errorf("store: kubernetes backend not yet implemented (roadmap phase 2)")
	default:
		return nil, fmt.Errorf("store: unknown backend %q", cfg.Backend)
	}
}
