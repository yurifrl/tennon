package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yurifrl/tennon/domain"
)

// Postgres persists resources as rows in a Postgres database. Multiple processes
// can share the database, but watch notifications are broadcast in-process after
// each committed write — the same level-based semantics the controller relies on.
type Postgres struct {
	pool *pgxpool.Pool
	b    *broadcaster
}

// NewPostgres opens a Postgres-backed store using the provided DSN.
func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// Create the resources table if it doesn't exist
	_, err = pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS resources (
  kind       TEXT NOT NULL,
  namespace  TEXT NOT NULL,
  name       TEXT NOT NULL,
  spec       BYTEA,
  status     BYTEA,
  labels     BYTEA,
  generation BIGINT NOT NULL DEFAULT 0,
  created    TIMESTAMPTZ NOT NULL,
  updated    TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (kind, namespace, name)
)`)
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Postgres{pool: pool, b: newBroadcaster()}, nil
}

func (p *Postgres) scan(row pgx.Row) (*domain.Resource, error) {
	var (
		r       domain.Resource
		spec    []byte
		status  []byte
		labels  []byte
		created time.Time
		updated time.Time
	)
	if err := row.Scan(&r.Kind, &r.Meta.Namespace, &r.Meta.Name, &spec, &status, &labels, &r.Meta.Generation, &created, &updated); err != nil {
		return nil, err
	}
	r.Spec = spec
	r.Status = status
	if len(labels) > 0 {
		_ = json.Unmarshal(labels, &r.Meta.Labels)
	}
	r.Meta.Created = created
	r.Meta.Updated = updated
	return &r, nil
}

func (p *Postgres) Get(ctx context.Context, kind, ns, name string) (*domain.Resource, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT kind,namespace,name,spec,status,labels,generation,created,updated FROM resources WHERE kind=$1 AND namespace=$2 AND name=$3`,
		kind, ns, name)
	r, err := p.scan(row)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return r, err
}

func (p *Postgres) List(ctx context.Context, kind, ns string) ([]*domain.Resource, error) {
	q := `SELECT kind,namespace,name,spec,status,labels,generation,created,updated FROM resources WHERE kind=$1`
	args := []any{kind}
	if ns != "" {
		q += ` AND namespace=$2`
		args = append(args, ns)
	}
	rows, err := p.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Resource
	for rows.Next() {
		var (
			r       domain.Resource
			spec    []byte
			status  []byte
			labels  []byte
			created time.Time
			updated time.Time
		)
		if err := rows.Scan(&r.Kind, &r.Meta.Namespace, &r.Meta.Name, &spec, &status, &labels, &r.Meta.Generation, &created, &updated); err != nil {
			return nil, err
		}
		r.Spec = spec
		r.Status = status
		if len(labels) > 0 {
			_ = json.Unmarshal(labels, &r.Meta.Labels)
		}
		r.Meta.Created = created
		r.Meta.Updated = updated
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (p *Postgres) Apply(ctx context.Context, r *domain.Resource) (*domain.Resource, error) {
	now := time.Now().UTC()
	labels, _ := json.Marshal(r.Meta.Labels)
	existing, err := p.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	evType := domain.WatchAdded
	created := now
	gen := int64(0)
	var status []byte
	if err == nil {
		evType = domain.WatchModified
		created = existing.Meta.Created
		gen = existing.Meta.Generation + 1
		status = existing.Status // Apply never clobbers status
	} else if err != domain.ErrNotFound {
		return nil, err
	}
	_, err = p.pool.Exec(ctx,
		`INSERT INTO resources(kind,namespace,name,spec,status,labels,generation,created,updated)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT(kind,namespace,name) DO UPDATE SET spec=excluded.spec,labels=excluded.labels,generation=excluded.generation,updated=excluded.updated`,
		r.Kind, r.Meta.Namespace, r.Meta.Name, []byte(r.Spec), status, labels, gen, created, now)
	if err != nil {
		return nil, err
	}
	stored, err := p.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return nil, err
	}
	p.b.broadcast(r.Kind, domain.WatchEvent{Type: evType, Resource: stored})
	return stored, nil
}

func (p *Postgres) UpdateStatus(ctx context.Context, r *domain.Resource) error {
	res, err := p.pool.Exec(ctx,
		`UPDATE resources SET status=$1, updated=$2 WHERE kind=$3 AND namespace=$4 AND name=$5`,
		[]byte(r.Status), time.Now().UTC(), r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	stored, err := p.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}
	p.b.broadcast(r.Kind, domain.WatchEvent{Type: domain.WatchModified, Resource: stored})
	return nil
}

func (p *Postgres) Delete(ctx context.Context, kind, ns, name string) error {
	existing, err := p.Get(ctx, kind, ns, name)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `DELETE FROM resources WHERE kind=$1 AND namespace=$2 AND name=$3`, kind, ns, name)
	if err != nil {
		return err
	}
	p.b.broadcast(kind, domain.WatchEvent{Type: domain.WatchDeleted, Resource: existing})
	return nil
}

func (p *Postgres) Watch(ctx context.Context, kind string) (<-chan domain.WatchEvent, func(), error) {
	ch, stop := p.b.subscribe(kind)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ch, stop, nil
}

func (p *Postgres) Close() error {
	p.pool.Close()
	return nil
}
