package main

import (
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
