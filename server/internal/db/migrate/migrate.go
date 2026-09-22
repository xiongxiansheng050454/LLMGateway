// Package migrate runs PostgreSQL schema migrations with tern.
package migrate

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	tern "github.com/jackc/tern/v2/migrate"
)

const versionTable = "public.schema_version"

// Run applies every pending migration in dir.
func Run(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if pool == nil {
		return fmt.Errorf("migrate: nil pool")
	}

	acquired, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer acquired.Release()

	migrator, err := tern.NewMigrator(ctx, acquired.Conn(), versionTable)
	if err != nil {
		return fmt.Errorf("migrate: initialize tern: %w", err)
	}
	if err := migrator.LoadMigrations(os.DirFS(dir)); err != nil {
		return fmt.Errorf("migrate: load migrations: %w", err)
	}
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate: run tern: %w", err)
	}
	return nil
}
