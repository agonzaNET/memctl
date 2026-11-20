package gitlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Timing of the committer (SPEC-0008 R4, R10).
const (
	QuietWindow    = 2 * time.Second
	MaxDelay       = 30 * time.Second
	LockRetries    = 3
	LockRetryDelay = time.Second
)

// Author is the git identity of tool commits, used as author and committer
// (SPEC-0008 R7).
type Author struct{ Name, Email string }

// DefaultAuthor is the identity when --git-author is not set.
var DefaultAuthor = Author{Name: "memctl", Email: "memctl@localhost"}

func (a Author) String() string { return a.Name + " <" + a.Email + ">" }

// ErrBadAuthor is returned by ParseAuthor for a value that breaks the rule.
var ErrBadAuthor = errors.New("gitlog: the git author must be Name <email>")

// ParseAuthor reads "Name <email>": a name of 1 to 100 characters, an email
// with exactly one @, no line break (SPEC-0008, "Inputs").
func ParseAuthor(s string) (Author, error) {
	if strings.ContainsAny(s, "\r\n") || !strings.HasSuffix(s, ">") {
		return Author{}, ErrBadAuthor
	}
	i := strings.LastIndex(s, " <")
	if i < 0 {
		return Author{}, ErrBadAuthor
	}
	name := strings.TrimSpace(s[:i])
	email := s[i+2 : len(s)-1]
	if n := utf8.RuneCountInString(name); n < 1 || n > 100 || strings.ContainsAny(name, "<>") {
		return Author{}, ErrBadAuthor
	}
	at := strings.Index(email, "@")
	if strings.Count(email, "@") != 1 || at == 0 || at == len(email)-1 || strings.ContainsAny(email, "<> \t") {
		return Author{}, ErrBadAuthor
	}
	return Author{Name: name, Email: email}, nil
}

// Entry is one successful tool write: the tool, the note title and the
// changed paths relative to the notes root, the old path first for a move
// (SPEC-0008 R3).
type Entry struct {
	Tool, Title string
	Paths       []string
}

// Clock is the time source of the committer's timing.
type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Config configures a Committer.
type Config struct {
	Root   string // the resolved notes root
	Author Author
	Log    *slog.Logger
	Clock  Clock // nil: the real clock
}

// Committer commits tool writes in the background: a batch is committed
// after a quiet window or a maximum delay (SPEC-0008).
type Committer struct {
	root    string
	author  Author
	log     *slog.Logger
	clock   Clock
	enabled bool

	mu          sync.Mutex
	batch       []Entry
	first, last time.Time
	notify      chan struct{}

	onIdle  func()              // test hook: the loop is about to wait
	sleep   func(time.Duration) // between index.lock retries
	timeout time.Duration       // of each git call
}

// New tests the root with git (SPEC-0008 R1) and returns a committer. When
// the root is not in a git work tree, or git cannot be run, it logs one WARN
// line and the committer drops every batch.
func New(ctx context.Context, cfg Config) *Committer {
	c := &Committer{
		root:    cfg.Root,
		author:  cfg.Author,
		log:     cfg.Log,
		clock:   cfg.Clock,
		notify:  make(chan struct{}, 1),
		sleep:   time.Sleep,
		timeout: GitTimeout,
	}
	if c.clock == nil {
		c.clock = realClock{}
	}
	ok, err := IsRepo(ctx, cfg.Root)
	switch {
	case err != nil:
		c.log.Warn("git cannot be run; tool changes are not committed")
	case !ok:
		c.log.Warn("the notes root is not in a git work tree; tool changes are not committed")
	default:
		c.enabled = true
	}
	return c
}

// Add queues one tool write without blocking (SPEC-0008 R3).
func (c *Committer) Add(e Entry) {
	e.Paths = append([]string(nil), e.Paths...)
	now := c.clock.Now()
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.first = now
	}
	c.last = now
	c.batch = append(c.batch, e)
	c.mu.Unlock()
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

// Run commits batches until ctx is cancelled; then it commits the pending
// batch at once, without waiting for the quiet window, and returns when that
// commit returns (SPEC-0008 R4, R12).
func (c *Committer) Run(ctx context.Context) {
	for {
		c.mu.Lock()
		pending := len(c.batch) > 0
		deadline := c.last.Add(QuietWindow)
		if max := c.first.Add(MaxDelay); max.Before(deadline) {
			deadline = max
		}
		c.mu.Unlock()

		var timer <-chan time.Time
		if pending {
			wait := deadline.Sub(c.clock.Now())
			if wait <= 0 {
				c.commitPending()
				continue
			}
			timer = c.clock.After(wait)
		}
		if c.onIdle != nil {
			c.onIdle()
		}
		select {
		case <-ctx.Done():
			c.commitPending()
			return
		case <-c.notify:
		case <-timer:
		}
	}
}

// commitPending takes the queued batch and commits it.
func (c *Committer) commitPending() {
	c.mu.Lock()
	batch := c.batch
	c.batch = nil
	c.mu.Unlock()
	if len(batch) == 0 || !c.enabled {
		return
	}
	c.commit(batch)
}

// message is the commit message of a batch: paths, not titles, and the old
// and the new path of a move (SPEC-0008 R5).
func message(entries []Entry) string {
	if len(entries) == 1 {
		return "memory: " + describeEntry(entries[0]) + "\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "memory: %d changes\n\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(&b, "- %s\n", describeEntry(e))
	}
	return b.String()
}

// describeEntry is "<tool> <path>", or "note_move <old> → <new>" for a move
// that changed the path.
func describeEntry(e Entry) string {
	switch {
	case len(e.Paths) == 0:
		return e.Tool
	case e.Tool == "note_move" && len(e.Paths) == 2 && e.Paths[0] != e.Paths[1]:
		return e.Tool + " " + e.Paths[0] + " → " + e.Paths[1]
	default:
		return e.Tool + " " + e.Paths[len(e.Paths)-1]
	}
}

// commit commits the paths of one batch, and only those (SPEC-0008 R6–R11).
func (c *Committer) commit(batch []Entry) {
	ctx := context.Background()
	paths := c.commitPaths(ctx, batch)
	if len(paths) == 0 {
		return
	}
	fail := func(err error) {
		var ge *gitError
		status, class := -1, classGitError
		if errors.As(err, &ge) {
			status, class = ge.status, ge.class
		}
		c.log.Error("commit failed", slog.Int("status", status), slog.String("class", class), slog.Int("paths", len(paths)))
	}

	add := append([]string{"-A", "--"}, paths...)
	if _, err := c.retry(func() ([]byte, error) { return runGit(ctx, c.root, c.timeout, nil, nil, "add", add...) }); err != nil {
		fail(err)
		return
	}
	env := []string{
		"GIT_AUTHOR_NAME=" + c.author.Name, "GIT_AUTHOR_EMAIL=" + c.author.Email,
		"GIT_COMMITTER_NAME=" + c.author.Name, "GIT_COMMITTER_EMAIL=" + c.author.Email,
	}
	msg := message(batch)
	commit := append([]string{"--only", "--cleanup=whitespace", "-F", "-", "--"}, paths...)
	_, err := c.retry(func() ([]byte, error) {
		return runGit(ctx, c.root, c.timeout, env, strings.NewReader(msg), "commit", commit...)
	})
	if err != nil {
		var ge *gitError
		if errors.As(err, &ge) && nothingToCommit(ge.out) {
			return // the files equal HEAD (SPEC-0008 R9)
		}
		fail(err)
		return
	}
	short, _ := runGit(ctx, c.root, c.timeout, nil, nil, "rev-parse", "--short", "HEAD")
	c.log.Info("committed", slog.Int("paths", len(paths)), slog.String("commit", strings.TrimSpace(string(short))))
}

// commitPaths returns the batch's paths in queue order without duplicates,
// without paths git ignores, and without paths that exist neither on disk
// nor in the index (a note created and removed within the batch).
func (c *Committer) commitPaths(ctx context.Context, batch []Entry) []string {
	seen := map[string]bool{}
	var paths []string
	for _, e := range batch {
		for _, p := range e.Paths {
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	stdin := strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	out, err := runGit(ctx, c.root, c.timeout, nil, stdin, "check-ignore", "-z", "--stdin")
	ignored := map[string]bool{}
	var ge *gitError
	if err == nil || (errors.As(err, &ge) && ge.status == 1) {
		for _, p := range bytes.Split(out, []byte{0}) {
			if len(p) > 0 {
				ignored[string(p)] = true
			}
		}
	}
	var keep []string
	for _, p := range paths {
		if ignored[p] {
			continue
		}
		if _, err := os.Lstat(filepath.Join(c.root, filepath.FromSlash(p))); errors.Is(err, fs.ErrNotExist) {
			if _, err := runGit(ctx, c.root, c.timeout, nil, nil, "rev-parse", "-q", "--verify", ":./"+p); err != nil {
				continue
			}
		}
		keep = append(keep, p)
	}
	return keep
}

// retry runs a git call again while it fails on a held index.lock, up to
// LockRetries more times, LockRetryDelay apart (SPEC-0008 R10).
func (c *Committer) retry(call func() ([]byte, error)) ([]byte, error) {
	out, err := call()
	for i := 0; i < LockRetries; i++ {
		var ge *gitError
		if !errors.As(err, &ge) || ge.class != classIndexLocked {
			break
		}
		c.sleep(LockRetryDelay)
		out, err = call()
	}
	return out, err
}

// nothingToCommit reports whether git's output (C locale) says the paths
// equal HEAD.
func nothingToCommit(out string) bool {
	return strings.Contains(out, "nothing to commit") || strings.Contains(out, "no changes added to commit") || strings.Contains(out, "nothing added to commit")
}
