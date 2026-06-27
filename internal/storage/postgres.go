package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaStatus reports whether required tables and migration-002 columns exist.
type SchemaStatus struct {
	TablesOK     bool
	Migration002 bool
}

func ConnectDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// VerifySchema checks that core tables and migration-002 columns are present.
func VerifySchema(ctx context.Context, pool *pgxpool.Pool) (SchemaStatus, error) {
	var status SchemaStatus

	var tableCount int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'transactions'
	`).Scan(&tableCount)
	if err != nil {
		return status, fmt.Errorf("check transactions table: %w", err)
	}
	status.TablesOK = tableCount > 0

	var columnCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'transactions' AND column_name = 'submission_kind'
	`).Scan(&columnCount)
	if err != nil {
		return status, fmt.Errorf("check migration 002 columns: %w", err)
	}
	status.Migration002 = columnCount > 0

	if !status.TablesOK {
		return status, fmt.Errorf("transactions table missing; run make migrate-up")
	}
	if !status.Migration002 {
		return status, fmt.Errorf("migration 002 columns missing; run make migrate-up")
	}
	return status, nil
}
