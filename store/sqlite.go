package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/yurifrl/tennon/domain"
	_ "modernc.org/sqlite"
)

// SQLite persists resources as rows in a local SQLite file. A single process
// owns the file, so watch notifications are broadcast in-process after each
// committed write — the same level-based semantics the controller relies on.
type SQLite struct {
	db *sql.DB
	b  *broadcaster
}

// NewSQLite opens (creating if needed) a SQLite-backed store at path.
func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite writer serialisation
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS resources (
  kind       TEXT NOT NULL,
  namespace  TEXT NOT NULL,
  name       TEXT NOT NULL,
  spec       BLOB,
  status     BLOB,
  labels     BLOB,
  generation INTEGER NOT NULL DEFAULT 0,
  created    TEXT NOT NULL,
  updated    TEXT NOT NULL,
  PRIMARY KEY (kind, namespace, name)
);`); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLite{db: db, b: newBroadcaster()}, nil
}

func (s *SQLite) scan(row interface{ Scan(...any) error }) (*domain.Resource, error) {
	var (
		r       domain.Resource
		spec    []byte
		status  []byte
		labels  []byte
		created string
		updated string
	)
	if err := row.Scan(&r.Kind, &r.Meta.Namespace, &r.Meta.Name, &spec, &status, &labels, &r.Meta.Generation, &created, &updated); err != nil {
		return nil, err
	}
	r.Spec = spec
	r.Status = status
	if len(labels) > 0 {
		_ = json.Unmarshal(labels, &r.Meta.Labels)
	}
	r.Meta.Created, _ = time.Parse(time.RFC3339Nano, created)
	r.Meta.Updated, _ = time.Parse(time.RFC3339Nano, updated)
	return &r, nil
}

func (s *SQLite) Get(ctx context.Context, kind, ns, name string) (*domain.Resource, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT kind,namespace,name,spec,status,labels,generation,created,updated FROM resources WHERE kind=? AND namespace=? AND name=?`,
		kind, ns, name)
	r, err := s.scan(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return r, err
}

func (s *SQLite) List(ctx context.Context, kind, ns string) ([]*domain.Resource, error) {
	q := `SELECT kind,namespace,name,spec,status,labels,generation,created,updated FROM resources WHERE kind=?`
	args := []any{kind}
	if ns != "" {
		q += ` AND namespace=?`
		args = append(args, ns)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Resource
	for rows.Next() {
		r, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLite) Apply(ctx context.Context, r *domain.Resource) (*domain.Resource, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	labels, _ := json.Marshal(r.Meta.Labels)
	existing, err := s.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	evType := domain.WatchAdded
	created := now
	gen := int64(0)
	var status []byte
	if err == nil {
		evType = domain.WatchModified
		created = existing.Meta.Created.Format(time.RFC3339Nano)
		gen = existing.Meta.Generation + 1
		status = existing.Status // Apply never clobbers status
	} else if err != domain.ErrNotFound {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO resources(kind,namespace,name,spec,status,labels,generation,created,updated)
		 VALUES(?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(kind,namespace,name) DO UPDATE SET spec=excluded.spec,labels=excluded.labels,generation=excluded.generation,updated=excluded.updated`,
		r.Kind, r.Meta.Namespace, r.Meta.Name, []byte(r.Spec), status, labels, gen, created, now); err != nil {
		return nil, err
	}
	stored, err := s.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return nil, err
	}
	s.b.broadcast(r.Kind, domain.WatchEvent{Type: evType, Resource: stored})
	return stored, nil
}

func (s *SQLite) UpdateStatus(ctx context.Context, r *domain.Resource) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE resources SET status=?, updated=? WHERE kind=? AND namespace=? AND name=?`,
		[]byte(r.Status), time.Now().UTC().Format(time.RFC3339Nano), r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	stored, err := s.Get(ctx, r.Kind, r.Meta.Namespace, r.Meta.Name)
	if err != nil {
		return err
	}
	s.b.broadcast(r.Kind, domain.WatchEvent{Type: domain.WatchModified, Resource: stored})
	return nil
}

func (s *SQLite) Delete(ctx context.Context, kind, ns, name string) error {
	existing, err := s.Get(ctx, kind, ns, name)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM resources WHERE kind=? AND namespace=? AND name=?`, kind, ns, name); err != nil {
		return err
	}
	s.b.broadcast(kind, domain.WatchEvent{Type: domain.WatchDeleted, Resource: existing})
	return nil
}

func (s *SQLite) Watch(ctx context.Context, kind string) (<-chan domain.WatchEvent, func(), error) {
	ch, stop := s.b.subscribe(kind)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ch, stop, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

// broadcaster is a small in-process fan-out used by stores that own their file.
type broadcaster struct {
	mu      sync.RWMutex
	watches map[string]map[int]chan domain.WatchEvent
	nextID  int
}

func newBroadcaster() *broadcaster {
	return &broadcaster{watches: map[string]map[int]chan domain.WatchEvent{}}
}

func (b *broadcaster) subscribe(kind string) (<-chan domain.WatchEvent, func()) {
	b.mu.Lock()
	if b.watches[kind] == nil {
		b.watches[kind] = map[int]chan domain.WatchEvent{}
	}
	id := b.nextID
	b.nextID++
	ch := make(chan domain.WatchEvent, 64)
	b.watches[kind][id] = ch
	b.mu.Unlock()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.watches[kind], id)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, stop
}

func (b *broadcaster) broadcast(kind string, ev domain.WatchEvent) {
	b.mu.RLock()
	subs := make([]chan domain.WatchEvent, 0, len(b.watches[kind]))
	for _, ch := range b.watches[kind] {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
