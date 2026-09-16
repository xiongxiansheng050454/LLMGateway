package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/db/migrate"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testEncryptionKey is a fixed 32-byte key used only by integration tests.
const testEncryptionKey = "0123456789abcdef0123456789abcdef"

// testStore returns a PostgreSQL-backed Store connected to TEST_DATABASE_URL,
// with migrations applied and channel data truncated. The test is skipped when
// TEST_DATABASE_URL is not set so `go test ./...` works without a database.
//
// The fixture is intentionally reusable by the user/usage store issues.
func testStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if err := migrate.Run(ctx, pool, filepath.Join("..", "..", "..", "db", "migrations")); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	// CASCADE clears dependent tables (channel_models, model_pricing,
	// client_api_keys, user_balances, balance_transactions, usage_logs) so the
	// fixture is reusable across domains.
	if _, err := pool.Exec(ctx, "TRUNCATE users, channels, rate_limit_rules RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate business tables: %v", err)
	}

	cipher, err := crypto.NewCipher([]byte(testEncryptionKey))
	if err != nil {
		t.Fatalf("build cipher: %v", err)
	}
	return New(pool, cipher)
}
