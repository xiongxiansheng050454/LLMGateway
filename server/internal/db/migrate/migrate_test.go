package migrate

import (
	"os"
	"testing"

	tern "github.com/jackc/tern/v2/migrate"
)

func TestLegacyMigrationNamesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(legacyMigrationNames))
	for _, name := range legacyMigrationNames {
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate legacy migration name %q", name)
		}
		seen[name] = struct{}{}
	}
}

func TestMigrationsLoadAsContiguousTernVersions(t *testing.T) {
	migrator, err := tern.NewMigrator(t.Context(), nil, versionTable)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	if err := migrator.LoadMigrations(os.DirFS("../../../db/migrations")); err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	if len(migrator.Migrations) != len(legacyMigrationNames) {
		t.Fatalf("migration count = %d, want %d", len(migrator.Migrations), len(legacyMigrationNames))
	}
}
