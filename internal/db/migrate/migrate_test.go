package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationFilesSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"000002_second.sql": "SELECT 2;",
		"000001_first.sql":  "SELECT 1;",
		"notes.txt":         "ignore me",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := MigrationFiles(dir)
	if err != nil {
		t.Fatalf("MigrationFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %v, want 2 sql files", files)
	}
	if filepath.Base(files[0]) != "000001_first.sql" || filepath.Base(files[1]) != "000002_second.sql" {
		t.Fatalf("files not sorted: %v", files)
	}
}

func TestMigrationFilesMissingDir(t *testing.T) {
	if _, err := MigrationFiles(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
