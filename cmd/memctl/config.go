package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

// mcpConfig is the configuration of memctl mcp (SPEC-0004, "Inputs").
type mcpConfig struct {
	root      string
	logLevel  string
	gitAuthor gitlog.Author
	scope     string // "" when none (SPEC-0011)
}

// authorRule is the rule of --git-author, named in its usage error
// (SPEC-0008, "Inputs").
const authorRule = "must be Name <email>: a name of 1 to 100 characters and an email with one @, on one line"

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// parseMCP reads the flags of memctl mcp and their environment variables; a
// flag wins over its variable. Errors name the flag or variable and the rule.
func parseMCP(args []string, getenv func(string) string) (mcpConfig, error) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("root", "", "")
	level := fs.String("log-level", "", "")
	author := fs.String("git-author", "", "")
	scope := fs.String("scope", "", "")
	if err := fs.Parse(args); err != nil {
		return mcpConfig{}, err
	}
	if fs.NArg() > 0 {
		return mcpConfig{}, errors.New("memctl mcp takes no positional arguments")
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	cfg := mcpConfig{root: *root, logLevel: *level}
	if !set["root"] {
		cfg.root = getenv("MEMCTL_ROOT")
	}
	if strings.TrimSpace(cfg.root) == "" {
		return mcpConfig{}, errors.New("--root (or MEMCTL_ROOT) is required")
	}
	if set["log-level"] {
		if _, ok := logLevels[cfg.logLevel]; !ok {
			return mcpConfig{}, errors.New("--log-level must be one of debug, info, warn, error")
		}
	} else {
		cfg.logLevel = getenv("MEMCTL_LOG_LEVEL")
		if cfg.logLevel == "" {
			cfg.logLevel = "info"
		}
		if _, ok := logLevels[cfg.logLevel]; !ok {
			return mcpConfig{}, errors.New("MEMCTL_LOG_LEVEL must be one of debug, info, warn, error")
		}
	}
	cfg.gitAuthor = gitlog.DefaultAuthor
	switch {
	case set["git-author"]:
		a, err := gitlog.ParseAuthor(*author)
		if err != nil {
			return mcpConfig{}, errors.New("--git-author " + authorRule)
		}
		cfg.gitAuthor = a
	case getenv("MEMCTL_GIT_AUTHOR") != "":
		a, err := gitlog.ParseAuthor(getenv("MEMCTL_GIT_AUTHOR"))
		if err != nil {
			return mcpConfig{}, errors.New("MEMCTL_GIT_AUTHOR " + authorRule)
		}
		cfg.gitAuthor = a
	}
	sc, err := parseScope(*scope, set["scope"], getenv)
	if err != nil {
		return mcpConfig{}, err
	}
	cfg.scope = sc
	return cfg, nil
}

// parseCheck reads the flags of memctl check (SPEC-0005, "Inputs").
func parseCheck(args []string, getenv func(string) string) (string, error) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("root", "", "")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", errors.New("memctl check takes no positional arguments")
	}
	set := false
	fs.Visit(func(f *flag.Flag) { set = set || f.Name == "root" })
	r := *root
	if !set {
		r = getenv("MEMCTL_ROOT")
	}
	if strings.TrimSpace(r) == "" {
		return "", errors.New("--root (or MEMCTL_ROOT) is required")
	}
	return r, nil
}
