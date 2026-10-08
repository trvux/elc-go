package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultMaxConns is the pool size when neither the DSN (pool_max_conns) nor
// DB_MAX_CONNS says otherwise. pgx's own default is max(4, NumCPU) — 4 on the
// 2-vCPU VPS — which nobody chose, and which production metrics showed to be
// the bottleneck: 66% of connection acquires had to queue (up to 13 requests
// in flight sharing 4 connections; RFC 2026-10-08, G4). Postgres allows 100
// connections and this service is its only heavy client.
const defaultMaxConns = 12

func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := poolConfig(dsn, os.Getenv)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// poolConfig builds the pool configuration. Split from New so the sizing rules
// can be tested without a database.
//
// Precedence for the pool size: a pool_max_conns parameter in the DSN wins
// (explicit and visible next to the connection string), then DB_MAX_CONNS,
// then defaultMaxConns. A malformed DB_MAX_CONNS is an error rather than a
// silent fallback: a typo should stop the deploy, not quietly run with a
// different pool than the operator thought they set.
func poolConfig(dsn string, getenv func(string) string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(dsn, "pool_max_conns") {
		maxConns := int32(defaultMaxConns)
		if raw := getenv("DB_MAX_CONNS"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				return nil, fmt.Errorf("DB_MAX_CONNS must be an integer between 1 and 100, got %q", raw)
			}
			maxConns = int32(n)
		}
		config.MaxConns = maxConns
	}

	// The DB is reached through Supabase's transaction-mode pooler (PgBouncer),
	// which does not preserve session state across pooled connections. pgx's
	// default mode caches named prepared statements per connection, which
	// collides under the pooler ("prepared statement already exists").
	// Simple protocol sends plain queries instead, which is what a
	// transaction-mode pooler requires. See ARCHITECTURE.md section 7.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	return config, nil
}
