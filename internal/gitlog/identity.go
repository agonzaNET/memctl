package gitlog

import (
	"context"
	"log/slog"
	"strings"
)

// repoIdentity returns the user.name and user.email that git reads in the
// root, each empty when it is not set (SPEC-0008 R14).
func (c *Committer) repoIdentity(ctx context.Context) (name, email string) {
	n, errN := runGit(ctx, c.root, c.timeout, nil, nil, "config", "--get", "user.name")
	e, errE := runGit(ctx, c.root, c.timeout, nil, nil, "config", "--get", "user.email")
	if errN == nil {
		name = strings.TrimSpace(string(n))
	}
	if errE == nil {
		email = strings.TrimSpace(string(e))
	}
	return name, email
}

// warnIdentity logs one WARN line when the repository lacks user.name or
// user.email, naming the missing keys and the identity that external
// changes get instead (SPEC-0008 R17).
func (c *Committer) warnIdentity(ctx context.Context) {
	name, email := c.repoIdentity(ctx)
	var missing []string
	if name == "" {
		missing = append(missing, "user.name")
	}
	if email == "" {
		missing = append(missing, "user.email")
	}
	if len(missing) == 0 {
		return
	}
	c.log.Warn("incomplete git identity",
		slog.String("missing", strings.Join(missing, ", ")),
		slog.String("author", c.author.String()))
}
