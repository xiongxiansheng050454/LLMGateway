package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreInterfacesDoNotReturnMapDTOs(t *testing.T) {
	files := []string{"channel.go", "channelhealth.go", "ratelimit.go", "usage.go", "user.go"}
	for _, name := range files {
		path := filepath.Join("..", "store", name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "map[string]any") {
			t.Fatalf("%s still exposes map[string]any in store contract", name)
		}
	}
}

func TestHTTPAPIDoesNotOwnRouteCandidate(t *testing.T) {
	path := filepath.Join("..", "httpapi", "openai_route.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if typeSpec.Name.Name == "RouteCandidate" {
				t.Fatalf("RouteCandidate must be defined in domain, not httpapi")
			}
		}
	}
}
