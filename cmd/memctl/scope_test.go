package main

import (
	"context"
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParseScope_SPEC0011_R1_R2_precedenceAndTrim(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		flagSet bool
		env     map[string]string
		want    string
	}{
		{"none", "", false, nil, ""},
		{"flag only", "work notes", true, nil, "work notes"},
		{"variable only", "", false, map[string]string{"MEMCTL_SCOPE": "personal notes"}, "personal notes"},
		{"flag wins", "work notes", true, map[string]string{"MEMCTL_SCOPE": "personal notes"}, "work notes"},
		{"empty flag turns off the variable", "", true, map[string]string{"MEMCTL_SCOPE": "personal notes"}, ""},
		{"surrounding white space", "  personal notes   ", true, nil, "personal notes"},
		{"only white space", "   ", true, nil, ""},
		{"variable trimmed", "", false, map[string]string{"MEMCTL_SCOPE": " personal "}, "personal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseScope(c.flag, c.flagSet, envOf(c.env))
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestParseScope_SPEC0011_R3_rules(t *testing.T) {
	max := strings.Repeat("ü", 200)
	if got, err := parseScope(max, true, envOf(nil)); err != nil || got != max {
		t.Fatalf("200 characters: %q, %v", got, err)
	}
	cases := []struct {
		name  string
		value string
	}{
		{"201 characters", strings.Repeat("ü", 201)},
		{"line break", "personal\nnotes"},
		{"carriage return", "personal\rnotes"},
		{"tab", "personal\tnotes"},
		{"other control character", "personal\x07notes"},
		{"invalid UTF-8", "personal \xff notes"},
	}
	for _, c := range cases {
		t.Run(c.name+" flag", func(t *testing.T) {
			_, err := parseScope(c.value, true, envOf(nil))
			if err == nil || !strings.HasPrefix(err.Error(), "--scope ") || !strings.Contains(err.Error(), "200 characters") {
				t.Fatalf("err %v, want the --scope rule", err)
			}
			if strings.Contains(err.Error(), "personal") || strings.Contains(err.Error(), "üü") {
				t.Fatalf("err echoes the value: %v", err)
			}
		})
		t.Run(c.name+" variable", func(t *testing.T) {
			_, err := parseScope("", false, envOf(map[string]string{"MEMCTL_SCOPE": c.value}))
			if err == nil || !strings.HasPrefix(err.Error(), "MEMCTL_SCOPE ") {
				t.Fatalf("err %v, want the MEMCTL_SCOPE rule", err)
			}
			if strings.Contains(err.Error(), "personal") {
				t.Fatalf("err echoes the value: %v", err)
			}
		})
	}
}

func TestUsage_SPEC0011_R1_namesTheFlag(t *testing.T) {
	if !strings.Contains(usageText, "--scope") || !strings.Contains(usageText, "MEMCTL_SCOPE") {
		t.Fatalf("usage does not name --scope and MEMCTL_SCOPE:\n%s", usageText)
	}
}

const scopeLine = "\nScope of this memory, set by the person who runs it: "

func TestCLI_SPEC0011_R3_badScopeExits2(t *testing.T) {
	root := notesRoot(t)
	r := runBin(t, binVersioned, nil, "mcp", "--root", root, "--scope", "secret\nline")
	if r.code != 2 || r.stdout != "" || !strings.Contains(r.stderr, "--scope") ||
		!strings.Contains(r.stderr, "200 characters") || strings.Contains(r.stderr, "secret") {
		t.Fatalf("bad --scope: %+v", r)
	}
	r = runBin(t, binVersioned, []string{"MEMCTL_SCOPE=" + strings.Repeat("x", 201)}, "mcp", "--root", root)
	if r.code != 2 || !strings.Contains(r.stderr, "MEMCTL_SCOPE") || strings.Contains(r.stderr, "xxxx") {
		t.Fatalf("bad MEMCTL_SCOPE: %+v", r)
	}
}

func TestCLI_SPEC0011_R4_startedScope(t *testing.T) {
	root := notesRoot(t)
	r := runBin(t, binVersioned, []string{"MEMCTL_SCOPE="}, "mcp", "--root", root, "--scope", "  work notes ")
	started := byMsg(jsonLines(t, r.stderr), "started")
	if r.code != 0 || started == nil || started["scope"] != "work notes" {
		t.Fatalf("with a scope: started %v; %s", started, r.stderr)
	}
	r = runBin(t, binVersioned, []string{"MEMCTL_SCOPE="}, "mcp", "--root", root)
	started = byMsg(jsonLines(t, r.stderr), "started")
	if _, has := started["scope"]; r.code != 0 || started == nil || has {
		t.Fatalf("without a scope: started %v; %s", started, r.stderr)
	}
}

func TestE2E_SPEC0011_R5_R6_binaryServesTheScopeLine(t *testing.T) {
	s := connectArgs(t, notesRoot(t), "--scope", "work notes")
	if got := s.cs.InitializeResult().Instructions; !strings.HasSuffix(got, scopeLine+"work notes") {
		t.Fatalf("instructions do not end with the scope line: %q", got)
	}
	res, err := s.cs.ListTools(context.Background(), nil)
	if err != nil || len(res.Tools) != 8 {
		t.Fatalf("tools: %v, %d", err, len(res.Tools))
	}
	for _, tool := range res.Tools {
		if !strings.HasSuffix(tool.Description, scopeLine+"work notes") {
			t.Errorf("%s description does not end with the scope line", tool.Name)
		}
	}
}
