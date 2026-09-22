package migrate

import (
	"os"
	"testing"

	tern "github.com/jackc/tern/v2/migrate"
)

func TestMigrationsLoadAsContiguousTernVersions(t *testing.T) {
	migrator, err := tern.NewMigrator(t.Context(), nil, versionTable)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	if err := migrator.LoadMigrations(os.DirFS("../../../db/migrations")); err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	for i, m := range migrator.Migrations {
		if m.Sequence != int32(i+1) {
			t.Fatalf("migration %d has sequence %d, want %d", i, m.Sequence, i+1)
		}
	}
}
