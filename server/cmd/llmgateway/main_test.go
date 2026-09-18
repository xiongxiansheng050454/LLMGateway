package main

import (
	"context"
	"strings"
	"testing"

	"LLMGateway/server/internal/config"
)

func TestBuildStoreRequiresDatabaseURL(t *testing.T) {
	st, closeStore, err := buildStore(context.Background(), config.Config{})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("buildStore error = %v, want missing DATABASE_URL error", err)
	}
	if st != nil || closeStore != nil {
		t.Fatalf("buildStore returned non-nil resources on invalid config: store=%t closeStore=%t", st != nil, closeStore != nil)
	}
}

func TestBuildStoreValidatesChannelKeyBeforeConnecting(t *testing.T) {
	st, closeStore, err := buildStore(context.Background(), config.Config{
		DatabaseURL: "postgres://127.0.0.1:1/unused?sslmode=disable",
	})
	if err == nil || !strings.Contains(err.Error(), "channel key encryption") {
		t.Fatalf("buildStore error = %v, want channel key encryption error", err)
	}
	if st != nil || closeStore != nil {
		t.Fatalf("buildStore returned non-nil resources on invalid config: store=%t closeStore=%t", st != nil, closeStore != nil)
	}
}
