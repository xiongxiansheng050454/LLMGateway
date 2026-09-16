package postgres

import (
	"testing"

	"LLMGateway/internal/store"
)

func TestNewImplementsStore(t *testing.T) {
	var st store.Store = New(nil)
	if st == nil {
		t.Fatal("New returned nil")
	}
}
