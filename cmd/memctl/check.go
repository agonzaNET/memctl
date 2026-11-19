package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/agonzaNET/memctl/internal/gitlog"
	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/parse"
)

// finding is one line of the memctl check report (SPEC-0005 R2, R4).
type finding struct {
	path string // "." for root-level findings
	line int    // 0 for none
	code string
	msg  string
}

// errorCodes are the findings with error severity (SPEC-0005 R3).
var errorCodes = map[string]bool{
	parse.CodeBadFrontmatter: true,
	parse.CodeInvalidUTF8:    true,
	"ambiguous_link":         true,
	"skipped_file":           true,
}

// runCheck runs memctl check (SPEC-0005 R1–R8).
func runCheck(args []string, stdout, stderr io.Writer) int {
	root, err := parseCheck(args, os.Getenv)
	if err != nil {
		return usageError(stderr, err.Error())
	}
	log := newLogger(stderr, "warn")
	svc, _, err := note.Open(root, log)
	if err != nil {
		log.Error("cannot open the notes root", slog.String("err", err.Error()))
		return exitError
	}

	var fs []finding
	err = svc.Call(func(idx *index.Index, _ note.Stats) error {
		for _, p := range idx.Paths() {
			n, _, _ := idx.Get(p)
			for _, pr := range n.Problems {
				fs = append(fs, finding{p, pr.Line, pr.Code, pr.Message})
			}
		}
		for _, l := range idx.Links() {
			switch r := l.Resolution; {
			case r.Status == index.Dangling:
				fs = append(fs, finding{l.Source, l.Line, "dangling_link",
					fmt.Sprintf("[[%s]] matches no note; create it or link to an existing note by path.", l.Target)})
			case r.Status == index.Ambiguous:
				fs = append(fs, finding{l.Source, l.Line, "ambiguous_link",
					fmt.Sprintf("[[%s]] matches several notes (%s); link to one of them by path.", l.Target, strings.Join(r.Candidates, ", "))})
			case r.Stale:
				fs = append(fs, finding{l.Source, l.Line, "stale_link",
					fmt.Sprintf("[[%s]] resolves only through an old path in the aliases of %s; link to %s instead.", l.Target, r.Path, strings.TrimSuffix(r.Path, ".md"))})
			}
		}
		return nil
	})
	if err != nil {
		log.Error("check failed", slog.String("class", "index"))
		return exitError
	}
	for p, reason := range svc.Skipped() {
		msg := "The note cannot be read and is not loaded; check its permissions."
		if reason == "too_large" {
			msg = "The note is larger than 4 MiB and is not loaded; split it into smaller notes."
		}
		fs = append(fs, finding{p, 0, "skipped_file", msg})
	}
	switch ok, err := gitlog.IsRepo(context.Background(), svc.Root()); {
	case errors.Is(err, gitlog.ErrGitMissing):
		fs = append(fs, finding{".", 0, "git_missing", "The git command cannot be run, so memctl cannot commit tool writes; install git."})
	case !ok:
		fs = append(fs, finding{".", 0, "no_git_repo", "The notes root is not in a git repository; memctl will not commit tool writes. Run git init in it."})
	}

	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if (a.path == ".") != (b.path == ".") {
			return b.path == "."
		}
		if a.path != b.path {
			return a.path < b.path
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return a.code < b.code
	})
	errs, warns := 0, 0
	for _, f := range fs {
		if errorCodes[f.code] {
			errs++
		} else {
			warns++
		}
		if f.line > 0 {
			fmt.Fprintf(stdout, "%s:%d: %s: %s\n", f.path, f.line, f.code, f.msg)
		} else {
			fmt.Fprintf(stdout, "%s: %s: %s\n", f.path, f.code, f.msg)
		}
	}
	notes := svc.Index().Len()
	fmt.Fprintf(stdout, "%d %s checked, %d %s, %d %s.\n",
		notes, plural(notes, "note", "notes"), errs, plural(errs, "error", "errors"), warns, plural(warns, "warning", "warnings"))
	if errs > 0 {
		return exitError
	}
	return exitOK
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
