package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("ADDR", "")
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
	t.Setenv("CHANNEL_MIN_ROUTE_BALANCE", "")
	t.Setenv(EnvChannelBreakerFailureThreshold, "")
	t.Setenv(EnvChannelBreakerCooldownSeconds, "")
	t.Setenv(EnvChannelBreakerWindowSeconds, "")
	t.Setenv(EnvChannelBreakerMinimumSamples, "")
	t.Setenv(EnvChannelBreakerErrorRatePercent, "")
	t.Setenv(EnvChannelBreakerTimeoutRatePercent, "")
	t.Setenv(EnvChannelBreakerBucketRetentionSeconds, "")

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
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
	if cfg.UpstreamTimeoutSeconds != 60 || cfg.UpstreamMaxAttempts != 3 || cfg.QuotaDefaultMaxTokens != 4096 || cfg.QuotaReservationTTLSeconds != 120 || cfg.QuotaReaperIntervalSeconds != 30 || cfg.QuotaReaperBatchSize != 100 || cfg.ChannelMinRouteBalance != "0.000000" {
		t.Fatalf("quota defaults = %+v", cfg)
	}
	if cfg.ChannelBreakerFailureThreshold != 5 || cfg.ChannelBreakerCooldownSeconds != 30 || cfg.ChannelBreakerWindowSeconds != 60 || cfg.ChannelBreakerMinimumSamples != 10 || cfg.ChannelBreakerErrorRatePercent != 50 || cfg.ChannelBreakerTimeoutRatePercent != 50 || cfg.ChannelBreakerBucketRetentionSeconds != 600 {
		t.Fatalf("breaker defaults = %+v", cfg)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("ADDR", ":9999")
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
	t.Setenv("CHANNEL_MIN_ROUTE_BALANCE", "2.5")
	t.Setenv(EnvChannelBreakerFailureThreshold, "9")
	t.Setenv(EnvChannelBreakerCooldownSeconds, "45")
	t.Setenv(EnvChannelBreakerWindowSeconds, "120")
	t.Setenv(EnvChannelBreakerMinimumSamples, "25")
	t.Setenv(EnvChannelBreakerErrorRatePercent, "80")
	t.Setenv(EnvChannelBreakerTimeoutRatePercent, "70")
	t.Setenv(EnvChannelBreakerBucketRetentionSeconds, "900")

	cfg := Load()
	if cfg.Addr != ":9999" {
		t.Fatalf("Addr = %q, want :9999", cfg.Addr)
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
	if cfg.UpstreamTimeoutSeconds != 90 || cfg.UpstreamMaxAttempts != 5 || cfg.QuotaDefaultMaxTokens != 8192 || cfg.QuotaReservationTTLSeconds != 180 || cfg.QuotaReaperIntervalSeconds != 15 || cfg.QuotaReaperBatchSize != 50 || cfg.ChannelMinRouteBalance != "2.500000" {
		t.Fatalf("quota overrides = %+v", cfg)
	}
	if cfg.ChannelBreakerFailureThreshold != 9 || cfg.ChannelBreakerCooldownSeconds != 45 || cfg.ChannelBreakerWindowSeconds != 120 || cfg.ChannelBreakerMinimumSamples != 25 || cfg.ChannelBreakerErrorRatePercent != 80 || cfg.ChannelBreakerTimeoutRatePercent != 70 || cfg.ChannelBreakerBucketRetentionSeconds != 900 {
		t.Fatalf("breaker overrides = %+v", cfg)
	}
}

func TestBreakerPercentRejectsOutOfRange(t *testing.T) {
	t.Setenv(EnvChannelBreakerErrorRatePercent, "150")
	t.Setenv(EnvChannelBreakerTimeoutRatePercent, "0")
	if cfg := Load(); cfg.ChannelBreakerErrorRatePercent != 50 || cfg.ChannelBreakerTimeoutRatePercent != 50 {
		t.Fatalf("percent clamps = %d/%d, want 50/50", cfg.ChannelBreakerErrorRatePercent, cfg.ChannelBreakerTimeoutRatePercent)
	}
}

func TestLoadMinimumRouteBalanceRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"-1", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("CHANNEL_MIN_ROUTE_BALANCE", value)
			if got := Load().ChannelMinRouteBalance; got != "0.000000" {
				t.Fatalf("ChannelMinRouteBalance = %q, want 0.000000", got)
			}
		})
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
