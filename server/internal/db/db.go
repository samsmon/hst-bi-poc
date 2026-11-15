// Package db runs read-only queries against PostgreSQL and returns rows in a
// JSON-friendly shape.
package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Result is one query result. Rows hold only JSON-encodable values.
type Result struct {
	Columns    []string `json:"columns"`
	Rows       [][]any  `json:"rows"`
	Truncated  bool     `json:"truncated"`
	DurationMS int64    `json:"duration_ms"`
}

// QueryError is a database error that is safe to show to the person who
// wrote the SQL (syntax errors, permission denied, statement timeout).
type QueryError struct {
	Message string
	Code    string
}

func (e *QueryError) Error() string { return e.Message }

// Timeout reports whether the statement was cancelled by a timeout.
func (e *QueryError) Timeout() bool { return e.Code == "57014" }

// DB is a small pool of read-only connections.
type DB struct{ pool *pgxpool.Pool }

// Open connects with the given URL. maxConns stays below the role's
// connection limit so the console cannot starve the dashboards.
func Open(ctx context.Context, url string, maxConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("DATABASE_URL is not a valid connection string")
	}
	cfg.MaxConns = maxConns
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close releases the pool.
func (d *DB) Close() { d.pool.Close() }

// Query runs sql in a read-only transaction and returns at most maxRows
// rows. Truncated is set when more rows were available.
func (d *DB) Query(ctx context.Context, sql string, args []any, maxRows int) (*Result, error) {
	start := time.Now()
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, wrap(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, wrap(err)
	}
	defer rows.Close()

	res := &Result{Rows: [][]any{}}
	for _, f := range rows.FieldDescriptions() {
		res.Columns = append(res.Columns, f.Name)
	}
	for rows.Next() {
		if len(res.Rows) >= maxRows {
			res.Truncated = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return nil, wrap(err)
		}
		for i, v := range vals {
			vals[i] = normalize(v)
		}
		res.Rows = append(res.Rows, vals)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, wrap(err)
	}
	res.DurationMS = time.Since(start).Milliseconds()
	return res, nil
}

func wrap(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return &QueryError{Message: pg.Message, Code: pg.Code}
	}
	return err
}
