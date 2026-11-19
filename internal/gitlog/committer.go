package gitlog

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Author is the git identity of tool commits.
type Author struct{ Name, Email string }

// ParseAuthor reads "Name <email>".
func ParseAuthor(s string) (Author, error) { return Author{}, errors.New("not implemented") }

// Entry is one tool write.
type Entry struct {
	Tool, Title string
	Paths       []string
}

// Clock is the time source of the committer.
type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

// Config configures a Committer.
type Config struct {
	Root   string
	Author Author
	Log    *slog.Logger
	Clock  Clock
}

// Committer commits tool writes.
type Committer struct{ onIdle func() }

// New returns a committer.
func New(ctx context.Context, cfg Config) *Committer { return &Committer{} }

// Add queues an entry.
func (c *Committer) Add(e Entry) {}

// Run runs the loop.
func (c *Committer) Run(ctx context.Context) { <-ctx.Done() }

func message(entries []Entry) string { return "" }
