package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestStoreInterfacesDoNotReturnMapDTOs(t *testing.T) {
	if _, err := filepath.Abs("."); err != nil {
		t.Fatal(err)
	}
}

func TestProxyDoesNotOwnRouteCandidate(t *testing.T) {
	path := filepath.Join("..", "proxy", "routing.go")
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
				t.Fatalf("RouteCandidate must be defined in catalog, not proxy")
			}
		}
	}
}
