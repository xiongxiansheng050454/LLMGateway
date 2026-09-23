package config

import (
	"os"
	"strconv"

	"LLMGateway/server/internal/money"
)

const EnvDatabaseURL = "DATABASE_URL"

// EnvChannelKey is the environment variable holding the channel api_key
// encryption key (raw bytes; 16, 24 or 32 bytes for AES-128/192/256). The
// configuration layer owns env parsing; crypto validates the key material.
const EnvChannelKey = "CHANNEL_KEY_ENCRYPTION_KEY"

const (
	EnvUpstreamRequestTimeout = "UPSTREAM_REQUEST_TIMEOUT"
	EnvUpstreamMaxAttempts    = "UPSTREAM_MAX_ATTEMPTS"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	MigrationsDir string
	// ChannelKeyEncryptionKey is the raw AES key used to encrypt upstream
	// channel api keys. It is required for every runtime configuration.
	ChannelKeyEncryptionKey string
	// UpstreamTimeoutSeconds bounds a downstream proxy call to the upstream
	// provider. Chat completions need far more than the default admin timeout.
	UpstreamTimeoutSeconds     int
	UpstreamMaxAttempts        int
	QuotaDefaultMaxTokens      int
	QuotaReservationTTLSeconds int
	QuotaReaperIntervalSeconds int
	QuotaReaperBatchSize       int
	ChannelMinRouteBalance     string
}

func Load() Config {
	cfg := Config{
		Addr:                    os.Getenv("ADDR"),
		DatabaseURL:             os.Getenv(EnvDatabaseURL),
		MigrationsDir:           os.Getenv("MIGRATIONS_DIR"),
		ChannelKeyEncryptionKey: os.Getenv(EnvChannelKey),
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "db/migrations"
	}
	cfg.UpstreamTimeoutSeconds = parsePositiveInt(os.Getenv(EnvUpstreamRequestTimeout), parsePositiveInt(os.Getenv("UPSTREAM_TIMEOUT_SECONDS"), 60))
	cfg.UpstreamMaxAttempts = parsePositiveInt(os.Getenv(EnvUpstreamMaxAttempts), 3)
	cfg.QuotaDefaultMaxTokens = parsePositiveInt(os.Getenv("QUOTA_DEFAULT_MAX_TOKENS"), 4096)
	cfg.QuotaReservationTTLSeconds = parsePositiveInt(os.Getenv("QUOTA_RESERVATION_TTL_SECONDS"), cfg.UpstreamTimeoutSeconds+60)
	if cfg.QuotaReservationTTLSeconds <= cfg.UpstreamTimeoutSeconds {
		cfg.QuotaReservationTTLSeconds = cfg.UpstreamTimeoutSeconds + 60
	}
	cfg.QuotaReaperIntervalSeconds = parsePositiveInt(os.Getenv("QUOTA_REAPER_INTERVAL_SECONDS"), 30)
	cfg.QuotaReaperBatchSize = parsePositiveInt(os.Getenv("QUOTA_REAPER_BATCH_SIZE"), 100)
	cfg.ChannelMinRouteBalance = parseNonNegativeAmount(os.Getenv("CHANNEL_MIN_ROUTE_BALANCE"), "0.000000")
	return cfg
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseNonNegativeAmount(value, fallback string) string {
	amount, err := money.Parse6(value)
	if value == "" || err != nil || amount.Cmp(0) < 0 {
		return fallback
	}
	return money.Format6(amount)
}
