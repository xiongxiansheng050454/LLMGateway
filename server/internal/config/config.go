package config

import (
	"os"
	"strconv"
)

const EnvDatabaseURL = "DATABASE_URL"

// EnvChannelKey is the environment variable holding the channel api_key
// encryption key (raw bytes; 16, 24 or 32 bytes for AES-128/192/256). The
// configuration layer owns env parsing; crypto validates the key material.
const EnvChannelKey = "CHANNEL_KEY_ENCRYPTION_KEY"

type Config struct {
	Addr          string
	DashboardDir  string
	DatabaseURL   string
	MigrationsDir string
	// ChannelKeyEncryptionKey is the raw AES key used to encrypt upstream
	// channel api keys. It is required for every runtime configuration.
	ChannelKeyEncryptionKey string
	// UpstreamTimeoutSeconds bounds a downstream proxy call to the upstream
	// provider. Chat completions need far more than the default admin timeout.
	UpstreamTimeoutSeconds int
}

func Load() Config {
	cfg := Config{
		Addr:                    os.Getenv("ADDR"),
		DashboardDir:            os.Getenv("DASHBOARD_DIR"),
		DatabaseURL:             os.Getenv(EnvDatabaseURL),
		MigrationsDir:           os.Getenv("MIGRATIONS_DIR"),
		ChannelKeyEncryptionKey: os.Getenv(EnvChannelKey),
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DashboardDir == "" {
		cfg.DashboardDir = "../dashboard"
	}
	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "db/migrations"
	}
	cfg.UpstreamTimeoutSeconds = parsePositiveInt(os.Getenv("UPSTREAM_TIMEOUT_SECONDS"), 60)
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
