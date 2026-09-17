package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("DASHBOARD_DIR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv(EnvChannelKey, "")

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.DashboardDir != "../dashboard" {
		t.Fatalf("DashboardDir = %q, want ../dashboard", cfg.DashboardDir)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.MigrationsDir != "db/migrations" {
		t.Fatalf("MigrationsDir = %q, want db/migrations", cfg.MigrationsDir)
	}
	if cfg.ChannelKeyEncryptionKey != "" {
		t.Fatalf("ChannelKeyEncryptionKey = %q, want empty", cfg.ChannelKeyEncryptionKey)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("ADDR", ":9999")
	t.Setenv("DASHBOARD_DIR", "/srv/dashboard")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("MIGRATIONS_DIR", "/srv/migrations")
	t.Setenv(EnvChannelKey, "0123456789abcdef0123456789abcdef")

	cfg := Load()
	if cfg.Addr != ":9999" {
		t.Fatalf("Addr = %q, want :9999", cfg.Addr)
	}
	if cfg.DashboardDir != "/srv/dashboard" {
		t.Fatalf("DashboardDir = %q, want /srv/dashboard", cfg.DashboardDir)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q, want override", cfg.DatabaseURL)
	}
	if cfg.MigrationsDir != "/srv/migrations" {
		t.Fatalf("MigrationsDir = %q, want /srv/migrations", cfg.MigrationsDir)
	}
	if cfg.ChannelKeyEncryptionKey != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("ChannelKeyEncryptionKey = %q, want override", cfg.ChannelKeyEncryptionKey)
	}
}

func TestUsePostgres(t *testing.T) {
	if (Config{}).UsePostgres() {
		t.Fatal("empty DatabaseURL should not enable PostgreSQL")
	}
	if !(Config{DatabaseURL: "postgres://localhost/db"}).UsePostgres() {
		t.Fatal("DatabaseURL should enable PostgreSQL")
	}
}
