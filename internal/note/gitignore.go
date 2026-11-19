package note

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// rule is one .gitignore pattern (SPEC-0003 R5).
type rule struct {
	re       *regexp.Regexp
	negate   bool
	dirOnly  bool
	anchored bool // matched against the path below the .gitignore's folder
}

// ignoreRules holds the .gitignore rules of every folder seen by a walk,
// keyed by the folder's slash path relative to the root ("" for the root).
type ignoreRules struct {
	byDir map[string][]rule
}

func newIgnoreRules() *ignoreRules { return &ignoreRules{byDir: map[string][]rule{}} }

// load reads the .gitignore of the folder dir (relative, "" for the root).
// A missing or unreadable file gives no rules; a bad line is skipped.
func (r *ignoreRules) load(root, dir string) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), ".gitignore"))
	if err != nil {
		return
	}
	var rules []rule
	for _, l := range strings.Split(string(data), "\n") {
		if ru, ok := parseRule(strings.TrimSuffix(l, "\r")); ok {
			rules = append(rules, ru)
		}
	}
	if len(rules) > 0 {
		r.byDir[dir] = rules
	}
}

// parseRule parses one .gitignore line with git's rules.
func parseRule(l string) (rule, bool) {
	// Trailing spaces are removed unless escaped with a backslash.
	for strings.HasSuffix(l, " ") && !strings.HasSuffix(l, `\ `) {
		l = l[:len(l)-1]
	}
	if l == "" || strings.HasPrefix(l, "#") {
		return rule{}, false
	}
	var ru rule
	if strings.HasPrefix(l, "!") {
		ru.negate = true
		l = l[1:]
	} else if strings.HasPrefix(l, `\!`) || strings.HasPrefix(l, `\#`) {
		l = l[1:]
	}
	if strings.HasSuffix(l, "/") {
		ru.dirOnly = true
		l = strings.TrimRight(l, "/")
	}
	if l == "" {
		return rule{}, false
	}
	if strings.Contains(l, "/") {
		ru.anchored = true
		l = strings.TrimPrefix(l, "/")
	}
	re, err := regexp.Compile("^" + globToRegexp(l) + "$")
	if err != nil {
		return rule{}, false
	}
	ru.re = re
	return ru, true
}

// globToRegexp converts a gitignore glob: "*" and "?" do not match "/",
// "[...]" is a character class, and "**" matches across folders.
func globToRegexp(g string) string {
	var b strings.Builder
	for i := 0; i < len(g); i++ {
		c := g[i]
		switch {
		case c == '\\' && i+1 < len(g):
			i++
			b.WriteString(regexp.QuoteMeta(string(g[i])))
		case strings.HasPrefix(g[i:], "**/") && (i == 0 || g[i-1] == '/'):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(g[i:], "**") && i+2 == len(g) && (i == 0 || g[i-1] == '/'):
			b.WriteString(".*")
			i++
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case c == '[':
			end := strings.IndexByte(g[i+1:], ']')
			if end < 0 {
				b.WriteString(regexp.QuoteMeta("["))
				continue
			}
			class := g[i+1 : i+1+end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + strings.ReplaceAll(class, `\`, `\\`) + "]")
			i += end + 1
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return b.String()
}

// ignored reports whether rel (a file, or a folder when isDir) is matched by
// the .gitignore files of the root and its folders. Deeper files are read
// later and the last matching rule wins (SPEC-0003 R5).
func (r *ignoreRules) ignored(rel string, isDir bool) bool {
	dirs := []string{""}
	parent := path.Dir(rel)
	if parent != "." {
		parts := strings.Split(parent, "/")
		for i := range parts {
			dirs = append(dirs, strings.Join(parts[:i+1], "/"))
		}
	}
	result := false
	for _, d := range dirs {
		below := rel
		if d != "" {
			below = strings.TrimPrefix(rel, d+"/")
		}
		for _, ru := range r.byDir[d] {
			if ru.dirOnly && !isDir {
				continue
			}
			subject := below
			if !ru.anchored {
				subject = path.Base(rel)
			}
			if ru.re.MatchString(subject) {
				result = !ru.negate
			}
		}
	}
	return result
}

// loadIgnoreRulesFor reads the .gitignore files of root and of every folder
// above rel.
func loadIgnoreRulesFor(root, rel string) *ignoreRules {
	r := newIgnoreRules()
	r.load(root, "")
	parent := path.Dir(rel)
	if parent != "." {
		parts := strings.Split(parent, "/")
		for i := range parts {
			r.load(root, strings.Join(parts[:i+1], "/"))
		}
	}
	return r
}
