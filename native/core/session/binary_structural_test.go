package session

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// This is a narrow structural guard for the two allocation regressions found
// in live review. Behavior tests remain authoritative for bytes and limits.
func TestMixedOutputAndLoadAvoidForbiddenCopies(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("output.go"))
	if err != nil {
		t.Fatal(err)
	}
	if containsMarshalJSONDecl(source) {
		t.Fatal("output.go reintroduced a per-entry custom MarshalJSON wrapper")
	}
	load, err := os.ReadFile(filepath.Join("load.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(load) == "" {
		t.Fatal("load.go source missing")
	}
	fileset := token.NewFileSet()
	parsed, err := parser.ParseFile(fileset, "load.go", load, 0)
	if err != nil {
		t.Fatal(err)
	}
	hasRawMessage := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel != nil && selector.Sel.Name == "RawMessage" {
			hasRawMessage = true
		}
		return true
	})
	if hasRawMessage {
		t.Fatal("load.go copies entry values through json.RawMessage")
	}
}

func containsMarshalJSONDecl(source []byte) bool {
	fileset := token.NewFileSet()
	parsed, err := parser.ParseFile(fileset, "output.go", source, 0)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		fn, ok := node.(*ast.FuncDecl)
		if ok && fn.Name.Name == "MarshalJSON" {
			found = true
		}
		return true
	})
	return found
}
