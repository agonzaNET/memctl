package parse_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestImports_SPEC0001_R23_pureParser: internal/parse imports only the
// standard library and yaml.v3, reads no files, logs nothing and uses no clock.
func TestImports_SPEC0001_R23_pureParser(t *testing.T) {
	forbidden := map[string]bool{"os": true, "io/fs": true, "io/ioutil": true, "log": true, "log/slog": true, "net": true, "net/http": true, "os/exec": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "time.Now(") {
			t.Errorf("%s uses the clock", f)
		}
		ast, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(p, "/")
			if strings.Contains(first, ".") && p != "gopkg.in/yaml.v3" {
				t.Errorf("%s imports %s", f, p)
			}
			if forbidden[p] {
				t.Errorf("%s imports %s", f, p)
			}
		}
	}
}
