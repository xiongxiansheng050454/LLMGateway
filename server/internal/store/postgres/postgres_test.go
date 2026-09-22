package postgres

import "testing"

func TestNewImplementsStore(t *testing.T) {
	if New(nil) == nil {
		t.Fatal("New returned nil")
	}
}
