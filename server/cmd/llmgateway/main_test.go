package main

import (
	"context"
	"testing"

	"LLMGateway/server/internal/config"
)

func TestBuildStoreWithoutDatabaseURLUsesMemory(t *testing.T) {
	st, closeStore, err := buildStore(context.Background(), config.Config{})
	if err != nil {
		t.Fatalf("buildStore: %v", err)
	}
	if st == nil {
		t.Fatal("buildStore returned nil store")
	}
	closeStore()

	if _, err := st.ListChannels(); err != nil {
		t.Fatalf("memory ListChannels: %v", err)
	}
}
