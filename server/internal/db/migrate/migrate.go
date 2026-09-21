// Package migrate runs PostgreSQL schema migrations with tern.
package migrate

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tern "github.com/jackc/tern/v2/migrate"
)

const (
	versionTable  = "public.schema_version"
	legacyTable   = "public.schema_migrations"
	bootstrapLock = int64(1876129820301756)
)

// legacyMigrationNames is the final sequence managed by the former runner.
// Tern requires unique, contiguous numeric versions, so its migration 10 is
// the file that previously shared the 000009 prefix.
var legacyMigrationNames = []string{
	"000001_init.sql",
	"000002_usage_logs_user_cascade.sql",
	"000003_balance_transactions_order_per_user.sql",
	"000004_drop_daily_usage_stats.sql",
	"000005_channel_health.sql",
	"000006_quota_reservations.sql",
	"000007_usage_logs_aggregate_indexes.sql",
	"000008_rate_limit_counters.sql",
	"000009_channel_breaker_windows.sql",
	"000009_remove_rate_limit_queue.sql",
}

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
	conn := acquired.Conn()

	if err := bootstrapLegacyVersion(ctx, conn); err != nil {
		return err
	}

	migrator, err := tern.NewMigrator(ctx, conn, versionTable)
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

func bootstrapLegacyVersion(ctx context.Context, conn *pgx.Conn) (err error) {
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", bootstrapLock); err != nil {
		return fmt.Errorf("migrate: acquire bootstrap lock: %w", err)
	}
	defer func() {
		_, unlockErr := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", bootstrapLock)
		if err == nil && unlockErr != nil {
			err = fmt.Errorf("migrate: release bootstrap lock: %w", unlockErr)
		}
	}()

	var ternTableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", versionTable).Scan(&ternTableExists); err != nil {
		return fmt.Errorf("migrate: check tern version table: %w", err)
	}
	if ternTableExists {
		return nil
	}

	var legacyTableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", legacyTable).Scan(&legacyTableExists); err != nil {
		return fmt.Errorf("migrate: check legacy version table: %w", err)
	}
	if !legacyTableExists {
		return nil
	}

	rows, err := conn.Query(ctx, "SELECT version FROM "+legacyTable)
	if err != nil {
		return fmt.Errorf("migrate: read legacy migration versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("migrate: scan legacy migration version: %w", err)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate: iterate legacy migration versions: %w", err)
	}

	if len(applied) == 0 {
		return nil
	}
	for _, name := range legacyMigrationNames {
		if _, ok := applied[name]; !ok {
			return fmt.Errorf("migrate: legacy schema_migrations is incomplete; missing %s", name)
		}
	}
	if len(applied) != len(legacyMigrationNames) {
		return fmt.Errorf("migrate: legacy schema_migrations contains unknown versions")
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate: begin tern version bootstrap: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "CREATE TABLE "+versionTable+" (version INT4 NOT NULL)"); err != nil {
		return fmt.Errorf("migrate: create tern version table: %w", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO "+versionTable+" (version) VALUES ($1)", len(legacyMigrationNames)); err != nil {
		return fmt.Errorf("migrate: bootstrap tern version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate: commit tern version bootstrap: %w", err)
	}
	return nil
}
