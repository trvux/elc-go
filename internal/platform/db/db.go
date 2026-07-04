package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// The DB is reached through Supabase's transaction-mode pooler (PgBouncer),
	// which does not preserve session state across pooled connections. pgx's
	// default mode caches named prepared statements per connection, which
	// collides under the pooler ("prepared statement already exists").
	// Simple protocol sends plain queries instead, which is what a
	// transaction-mode pooler requires. See ARCHITECTURE.md section 7.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

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
