package config

import (
	"os"
	"strconv"

	"LLMGateway/internal/crypto"
)

type Config struct {
	Addr          string
	DashboardDir  string
	DatabaseURL   string
	MigrationsDir string
	// ChannelKeyEncryptionKey is the raw AES key used to encrypt upstream
	// channel api keys. Required when DatabaseURL is set.
	ChannelKeyEncryptionKey string
	// UpstreamTimeoutSeconds bounds a downstream proxy call to the upstream
	// provider. Chat completions need far more than the default admin timeout.
	UpstreamTimeoutSeconds int
}

func Load() Config {
	cfg := Config{
		Addr:                    os.Getenv("ADDR"),
		DashboardDir:            os.Getenv("DASHBOARD_DIR"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		MigrationsDir:           os.Getenv("MIGRATIONS_DIR"),
		ChannelKeyEncryptionKey: os.Getenv(crypto.EnvChannelKey),
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DashboardDir == "" {
		cfg.DashboardDir = "dashboard"
	}
	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "db/migrations"
	}
	cfg.UpstreamTimeoutSeconds = parsePositiveInt(os.Getenv("UPSTREAM_TIMEOUT_SECONDS"), 60)
	return cfg
}

// UsePostgres reports whether a PostgreSQL store should be used instead of the
// default in-memory store.
func (c Config) UsePostgres() bool {
	return c.DatabaseURL != ""
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
