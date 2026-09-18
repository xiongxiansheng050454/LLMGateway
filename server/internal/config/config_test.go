package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("DASHBOARD_DIR", "")
	t.Setenv(EnvDatabaseURL, "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv(EnvChannelKey, "")
	t.Setenv("UPSTREAM_TIMEOUT_SECONDS", "")
	t.Setenv(EnvUpstreamRequestTimeout, "")
	t.Setenv(EnvUpstreamMaxAttempts, "")
	t.Setenv("QUOTA_DEFAULT_MAX_TOKENS", "")
	t.Setenv("QUOTA_RESERVATION_TTL_SECONDS", "")
	t.Setenv("QUOTA_REAPER_INTERVAL_SECONDS", "")
	t.Setenv("QUOTA_REAPER_BATCH_SIZE", "")

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
	if cfg.UpstreamTimeoutSeconds != 60 || cfg.UpstreamMaxAttempts != 3 || cfg.QuotaDefaultMaxTokens != 4096 || cfg.QuotaReservationTTLSeconds != 120 || cfg.QuotaReaperIntervalSeconds != 30 || cfg.QuotaReaperBatchSize != 100 {
		t.Fatalf("quota defaults = %+v", cfg)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("ADDR", ":9999")
	t.Setenv("DASHBOARD_DIR", "/srv/dashboard")
	t.Setenv(EnvDatabaseURL, "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("MIGRATIONS_DIR", "/srv/migrations")
	t.Setenv(EnvChannelKey, "0123456789abcdef0123456789abcdef")
	t.Setenv("UPSTREAM_TIMEOUT_SECONDS", "90")
	t.Setenv(EnvUpstreamRequestTimeout, "90")
	t.Setenv(EnvUpstreamMaxAttempts, "5")
	t.Setenv("QUOTA_DEFAULT_MAX_TOKENS", "8192")
	t.Setenv("QUOTA_RESERVATION_TTL_SECONDS", "180")
	t.Setenv("QUOTA_REAPER_INTERVAL_SECONDS", "15")
	t.Setenv("QUOTA_REAPER_BATCH_SIZE", "50")

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
	if cfg.UpstreamTimeoutSeconds != 90 || cfg.UpstreamMaxAttempts != 5 || cfg.QuotaDefaultMaxTokens != 8192 || cfg.QuotaReservationTTLSeconds != 180 || cfg.QuotaReaperIntervalSeconds != 15 || cfg.QuotaReaperBatchSize != 50 {
		t.Fatalf("quota overrides = %+v", cfg)
	}
}

func TestQuotaReservationTTLExceedsUpstreamTimeout(t *testing.T) {
	t.Setenv("UPSTREAM_TIMEOUT_SECONDS", "90")
	t.Setenv("QUOTA_RESERVATION_TTL_SECONDS", "60")
	cfg := Load()
	if cfg.QuotaReservationTTLSeconds != 150 {
		t.Fatalf("QuotaReservationTTLSeconds = %d, want 150", cfg.QuotaReservationTTLSeconds)
	}
}
