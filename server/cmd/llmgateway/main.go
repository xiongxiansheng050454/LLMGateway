package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"LLMGateway/server/internal/config"
	"LLMGateway/server/internal/crypto"
	"LLMGateway/server/internal/db/migrate"
	"LLMGateway/server/internal/httpapi"
	"LLMGateway/server/internal/store"
	"LLMGateway/server/internal/store/memory"
	"LLMGateway/server/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, closeStore, err := buildStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeStore()

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: newRouter(cfg.DashboardDir, st,
			httpapi.WithUpstreamTimeout(time.Duration(cfg.UpstreamTimeoutSeconds)*time.Second)),
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("LLMGateway listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Print("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// buildStore selects the persistence implementation from configuration.
func buildStore(ctx context.Context, cfg config.Config) (store.Store, func(), error) {
	if !cfg.UsePostgres() {
		log.Print("DATABASE_URL not set, using in-memory store")
		return memory.New(), func() {}, nil
	}

	cipher, err := crypto.NewCipher([]byte(cfg.ChannelKeyEncryptionKey))
	if err != nil {
		return nil, nil, fmt.Errorf("channel key encryption: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := migrate.Run(ctx, pool, cfg.MigrationsDir); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	log.Print("using PostgreSQL store")
	return postgres.New(pool, cipher), pool.Close, nil
}
