package arch_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// modulePath is this module's import path, from go.mod.
const modulePath = "github.com/agonzaNET/memctl"

// thirdParty is every third-party module memctl may use, and the one package
// that may import it (ADR-0010, ADR-0011). A module enters this map in the
// change that first imports it, with its line in go.mod; the MCP SDK joins
// with internal/mcpserver. A module outside ADR-0011 needs an ADR first.
var thirdParty = map[string]string{
	"gopkg.in/yaml.v3": "internal/parse",
}

// moduleRoot is the repository root, two levels up from this package.
var moduleRoot = filepath.Join("..", "..")

// ownerOf returns the module of thirdParty that path belongs to, or "".
func ownerOf(path string) string {
	for mod := range thirdParty {
		if path == mod || strings.HasPrefix(path, mod+"/") {
			return mod
		}
	}
	return ""
}

// isThirdParty reports whether an import path is neither the standard library
// nor this module.
func isThirdParty(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return strings.Contains(first, ".") && path != modulePath && !strings.HasPrefix(path, modulePath+"/")
}

// TestImports_ADR0010_eachModuleHasOneImporter: a third-party module is
// imported only by its package, in non-test code. Tests may drive an SDK (an
// MCP client in a test, for example). An import of a module that is not in
// thirdParty fails too, and so does a module in thirdParty that nothing
// imports.
func TestImports_ADR0010_eachModuleHasOneImporter(t *testing.T) {
	seen := map[string]bool{}
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "bin", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(moduleRoot, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !isThirdParty(p) {
				continue
			}
			mod := ownerOf(p)
			if mod == "" {
				t.Errorf("%s imports %s, which no ADR allows (add it to thirdParty with its ADR)", path, p)
				continue
			}
			seen[mod] = true
			if want := thirdParty[mod]; rel != want {
				t.Errorf("%s imports %s; only %s may (ADR-0010)", path, p, want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for mod, pkg := range thirdParty {
		if !seen[mod] {
			t.Errorf("%s is allowed for %s but nothing imports it: drop it from thirdParty and go.mod", mod, pkg)
		}
	}
}

// requireLine is one direct requirement of go.mod.
var requireLine = regexp.MustCompile(`^\s*(?:require\s+)?([a-z0-9.-]+\.[a-z]+/\S+)\s+v\S+\s*$`)

// TestImports_ADR0010_goModMatches: every direct requirement of go.mod is a
// module of thirdParty, so a module cannot come in through go.mod alone.
func TestImports_ADR0010_goModMatches(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "// indirect") {
			continue
		}
		m := requireLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if _, ok := thirdParty[m[1]]; !ok {
			t.Errorf("go.mod requires %s directly, which no ADR allows (add it to thirdParty with its ADR)", m[1])
		}
	}
}
