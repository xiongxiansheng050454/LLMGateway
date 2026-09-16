package postgres

import (
	"testing"

	"LLMGateway/server/internal/store"
)

func TestNewImplementsStore(t *testing.T) {
	var st store.Store = New(nil, nil)
	if st == nil {
		t.Fatal("New returned nil")
	}
}
