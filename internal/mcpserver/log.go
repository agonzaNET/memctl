package mcpserver

import (
	"log/slog"
	"sort"
	"time"
)

// callLog collects the counts of one tool call for its log line (ADR-0016).
// Tools only set counts with fixed names; no argument or note text.
type callLog struct {
	counts map[string]int
}

func (c *callLog) set(name string, n int) {
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	c.counts[name] = n
}

// write logs the one "tool call" line (SPEC-0004 R12).
func (c *callLog) write(log *slog.Logger, tool string, start time.Time, code Code) {
	attrs := []any{
		slog.String("tool", tool),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	}
	if code != "" {
		attrs = append(attrs, slog.String("error", string(code)))
	}
	names := make([]string, 0, len(c.counts))
	for n := range c.counts {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		attrs = append(attrs, slog.Int(n, c.counts[n]))
	}
	log.Info("tool call", attrs...)
}
