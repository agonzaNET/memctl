package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"
)

// mcpConfig is the configuration of memctl mcp (SPEC-0004, "Inputs").
type mcpConfig struct {
	root     string
	logLevel string
}

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
	return cfg, nil
}
