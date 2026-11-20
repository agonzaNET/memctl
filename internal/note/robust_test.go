package note_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/agonzaNET/memctl/internal/gitlog"
	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

// TestRobust_SPEC0007_R1_R2_R3_faultsInEveryWrite injects a fault before
// the rename, and a human edit before the re-hash, into each write tool's
// operation: the notes stay as they were, no temporary file is left, the
// index equals a fresh load, and nothing goes to the committer.
func TestRobust_SPEC0007_R1_R2_R3_faultsInEveryWrite(t *testing.T) {
	const a = "---\ntitle: A\n---\n# S\nbody\n"
	ops := map[string]func(s *note.Service, sha string) error{
		"write": func(s *note.Service, _ string) error {
			_, err := s.Write(note.NewNote{Title: "New", Folder: "d", Body: "x"})
			return err
		},
		"edit": func(s *note.Service, sha string) error {
			_, err := s.Edit(resolvePath("d/a.md"), note.EditOp{ExpectedSHA: sha, Op: "append", Text: "x"})
			return err
		},
		"move": func(s *note.Service, _ string) error {
			_, err := s.Move(resolvePath("d/a.md"), note.MoveOp{NewPath: "e/b.md"})
			return err
		},
		"delete": func(s *note.Service, _ string) error {
			_, err := s.Delete(resolvePath("d/a.md"))
			return err
		},
	}
	boom := errors.New("injected")
	for name, op := range ops {
		for _, fault := range []string{"rename", "rehash"} {
			t.Run(name+" "+fault, func(t *testing.T) {
				root := tree(t, map[string]string{"d/a.md": a})
				rec := &recCommitter{}
				s := openWith(t, root, fixedClock, note.WithCommitter(rec))
				want := a
				if fault == "rename" {
					note.SetHooks(s, nil, func(string) error { return boom })
				} else {
					want = a + "human\n"
					note.SetHooks(s, func(abs string) {
						if strings.HasSuffix(abs, "a.md") || strings.HasSuffix(abs, "New.md") {
							os.WriteFile(filepath.Join(root, "d", "a.md"), []byte(want), 0o644)
						}
					}, nil)
				}
				err := op(s, shaOf(t, root, "d/a.md"))
				if fault == "rename" && !errors.Is(err, boom) {
					t.Fatalf("err %v, want the injected fault", err)
				}
				if fault == "rehash" && name != "write" && !errors.Is(err, note.ErrConflict) {
					t.Fatalf("err %v, want conflict", err)
				}
				if name == "write" && fault == "rehash" {
					return // the new note's path is untouched by the edit of a.md
				}
				note.SetHooks(s, nil, nil)
				if got := readFile(t, root, "d/a.md"); got != want {
					t.Fatalf("d/a.md is %q, want %q", got, want)
				}
				for _, p := range []string{"d/New.md", "e/b.md"} {
					if _, err := os.Stat(filepath.Join(root, p)); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("%s exists after a failed %s", p, name)
					}
				}
				for _, d := range []string{root, filepath.Join(root, "d"), filepath.Join(root, "e")} {
					if _, err := os.Stat(d); err == nil {
						if left := dotFiles(t, d); len(left) != 0 {
							t.Fatalf("temporary files left in %s: %v", d, left)
						}
					}
				}
				if len(rec.changes()) != 0 {
					t.Fatalf("a failed write handed %+v", rec.changes())
				}
				assertIndexMatchesDisk(t, s) // the human edit is picked up by the next check
			})
		}
	}
}

// TestRobust_SPEC0007_R4_R5_randomSequence runs random tool writes mixed
// with editor changes. After each step the index equals a fresh load, and
// the committer has received exactly the paths of the successful writes.
func TestRobust_SPEC0007_R4_R5_randomSequence(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			root := t.TempDir()
			rec := &recCommitter{}
			s := openWith(t, root, fixedClock, note.WithCommitter(rec))
			var want []note.Change
			titles := []string{"Alpha", "Beta", "Gamma", "Delta"}
			folders := []string{"", "x", "x/y"}
			pick := func() (string, bool) {
				var ps []string
				s.Call(func(idx *index.Index, _ note.Stats) error { ps = idx.Paths(); return nil })
				if len(ps) == 0 {
					return "", false
				}
				return ps[rng.Intn(len(ps))], true
			}
			for step := 0; step < 120; step++ {
				editor := false
				switch k := rng.Intn(7); k {
				case 0, 1:
					title := titles[rng.Intn(len(titles))]
					w, err := s.Write(note.NewNote{Title: title, Folder: folders[rng.Intn(len(folders))], Body: "links [[Alpha]]"})
					if err == nil {
						want = append(want, note.Change{Tool: "note_write", Title: title, Paths: []string{w.Path}})
					}
				case 2:
					if p, ok := pick(); ok {
						e, err := s.Edit(resolvePath(p), note.EditOp{ExpectedSHA: shaOf(t, root, p), Op: "append", Text: fmt.Sprint("line ", step)})
						if err != nil {
							t.Fatalf("step %d edit %s: %v", step, p, err)
						}
						want = append(want, note.Change{Tool: "note_edit", Title: e.Title, Paths: []string{p}})
					}
				case 3:
					if p, ok := pick(); ok {
						to := folders[rng.Intn(len(folders))]
						if to != "" {
							to += "/"
						}
						mv, err := s.Move(resolvePath(p), note.MoveOp{NewPath: fmt.Sprintf("%sm%d.md", to, step)})
						if err == nil && !mv.Same {
							want = append(want, note.Change{Tool: "note_move", Title: mv.Title, Paths: []string{p, mv.Path}})
						}
					}
				case 4:
					if p, ok := pick(); ok {
						d, err := s.Delete(resolvePath(p))
						if err != nil {
							t.Fatalf("step %d delete %s: %v", step, p, err)
						}
						want = append(want, note.Change{Tool: "note_delete", Title: d.Title, Paths: []string{p}})
					}
				case 5: // an editor writes or changes a note
					editor = true
					p := fmt.Sprintf("%se%d.md", map[bool]string{true: "x/", false: ""}[rng.Intn(2) == 0], rng.Intn(4))
					os.MkdirAll(filepath.Join(root, filepath.Dir(p)), 0o755)
					// Saves of the same size, often within one timestamp
					// tick: SPEC-0003 R17 makes the next check see them.
					os.WriteFile(filepath.Join(root, p), []byte(fmt.Sprintf("---\ntitle: E%03d\n---\nstep %03d\n", step, step)), 0o644)
				case 6: // an editor deletes a note
					editor = true
					if p, ok := pick(); ok {
						os.Remove(filepath.Join(root, p))
					}
				}
				if editor {
					assertIndexMatchesDisk(t, s)
				} else {
					assertIndexFresh(t, s)
				}
				if got := rec.changes(); !reflect.DeepEqual(got, want) {
					t.Fatalf("step %d: committer got %d changes, want %d:\n%+v\n%+v", step, len(got), len(want), got, want)
				}
			}
		})
	}
}

// both forwards each change to the git committer and records it.
type both struct {
	rec *recCommitter
	git *gitlog.Committer
}

func (b both) Enqueue(c note.Change) {
	b.rec.Enqueue(c)
	b.git.Add(gitlog.Entry{Tool: c.Tool, Title: c.Title, Paths: c.Paths})
}

func (b both) External(paths []string) {
	b.rec.External(paths)
	b.git.External(paths)
}

func (b both) WaitExternal(path string) error { return b.git.WaitExternal(path) }

// TestRobust_SPEC0007_R1_concurrentCalls runs tool writes and reads from
// several goroutines while the git committer runs (run with -race). Every
// change reaches git once the committer stops.
func TestRobust_SPEC0007_R1_concurrentCalls(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	log := logger(new(bytes.Buffer))
	gc := gitlog.New(context.Background(), gitlog.Config{Root: root, Author: gitlog.DefaultAuthor, Log: log})
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { gc.Run(ctx); close(stopped) }()
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(both{rec, gc}))

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				w, err := s.Write(note.NewNote{Title: fmt.Sprintf("n%d-%d", g, i), Body: "b"})
				if err != nil {
					t.Error(err)
					return
				}
				if _, err := s.Edit(resolvePath(w.Path), note.EditOp{ExpectedSHA: w.SHA256, Op: "append", Text: "more"}); err != nil {
					t.Error(err)
					return
				}
				s.Call(func(idx *index.Index, _ note.Stats) error {
					idx.Search(index.Query{Text: "more", Limit: 5})
					return nil
				})
			}
		}(g)
	}
	wg.Wait()
	cancel()
	<-stopped
	if n := len(rec.changes()); n != 8*10*2 {
		t.Fatalf("%d changes, want %d", n, 8*10*2)
	}
	assertIndexFresh(t, s)
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "?? .memory/" && len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("uncommitted after the committer stopped: %v %q", err, out)
	}
}
