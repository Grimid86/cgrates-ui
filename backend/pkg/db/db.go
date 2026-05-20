// Package db provides PostgreSQL connection pooling with support for
// read replicas, tenant context, and prepared statements.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool with additional utilities
type Pool struct {
	primary *pgxpool.Pool
	replica *pgxpool.Pool // optional hot standby
}

// Config for database connections
type Config struct {
	DSN        string
	MaxConns   int32
	MinConns   int32
	MaxLifetime time.Duration
	MaxIdleTime time.Duration
}

// New creates a new database pool
func New(cfg Config) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	} else {
		poolCfg.MaxConns = 10
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	poolCfg.MaxConnLifetime = cfg.MaxLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Pool{primary: pool}, nil
}

// Close closes all database connections
func (p *Pool) Close() {
	if p.primary != nil {
		p.primary.Close()
	}
	if p.replica != nil {
		p.replica.Close()
	}
}

// Primary returns the primary (write) connection pool
func (p *Pool) Primary() *pgxpool.Pool {
	return p.primary
}

// QueryRow executes a query that returns a single row on the primary
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.primary.QueryRow(ctx, sql, args...)
}

// Query executes a query on the primary
func (p *Pool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.primary.Query(ctx, sql, args...)
}

// Exec executes a command on the primary
func (p *Pool) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := p.primary.Exec(ctx, sql, args...)
	return err
}

// Transaction runs a function within a database transaction
func (p *Pool) Transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := p.primary.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetTenantContext sets the tenant_id and role for RLS policies
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID int64, role string) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	_, err = tx.Exec(ctx, "SET LOCAL app.current_role = $1", role)
	if err != nil {
		return fmt.Errorf("set role context: %w", err)
	}
	return nil
}

// SetUserContext sets the user_id and IP for audit triggers
func SetUserContext(ctx context.Context, tx pgx.Tx, userID int64, portalType string, ip string) error {
	_, err := tx.Exec(ctx, "SET LOCAL app.current_user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("set user context: %w", err)
	}
	_, err = tx.Exec(ctx, "SET LOCAL app.current_portal = $1", portalType)
	if err != nil {
		return fmt.Errorf("set portal context: %w", err)
	}
	if ip != "" {
		_, err = tx.Exec(ctx, "SET LOCAL app.current_ip = $1", ip)
		if err != nil {
			return fmt.Errorf("set ip context: %w", err)
		}
	}
	return nil
}
