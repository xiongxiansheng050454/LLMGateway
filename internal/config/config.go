package config

import (
	"os"

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
	return cfg
}

// UsePostgres reports whether a PostgreSQL store should be used instead of the
// default in-memory store.
func (c Config) UsePostgres() bool {
	return c.DatabaseURL != ""
}
