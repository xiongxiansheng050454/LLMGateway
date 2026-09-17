package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisibleBackendModuleDirectories(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{
		"internal/httpapi",
		"internal/protocol/openai",
		"internal/domain",
		"internal/store",
		"internal/store/memory",
		"internal/store/postgres",
		"internal/db/sqlc",
	} {
		if info, err := os.Stat(filepath.Join(root, dir)); err != nil || !info.IsDir() {
			t.Fatalf("expected module directory %s to exist", dir)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "internal/handler")); !os.IsNotExist(err) {
		t.Fatalf("internal/handler should not remain as the HTTP catch-all module")
	}
}

func TestHTTPRouteTableStaysCentralized(t *testing.T) {
	root := filepath.Join("..", "..")
	content, err := os.ReadFile(filepath.Join(root, "cmd", "llmgateway", "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/healthz", "/admin/", "/v1/", "/dashboard/"} {
		if !strings.Contains(string(content), route) {
			t.Fatalf("cmd/llmgateway/router.go is missing route %s", route)
		}
	}
}

func TestOpenAIWireTypesStayInProtocolPackage(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{"internal/domain", "internal/store"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(content), "OpenAIModel") || strings.Contains(string(content), "ChatCompletionRequest") {
				t.Fatalf("OpenAI wire type leaked into %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
