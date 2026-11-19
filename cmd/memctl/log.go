package main

import (
	"io"
	"log/slog"
)

// newLogger returns the JSON logger on stderr at the given level (SPEC-0004
// R4). stdout is never used for logs.
func newLogger(stderr io.Writer, level string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: logLevels[level]}))
}
