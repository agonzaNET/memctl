package note

import (
	"errors"
	"log/slog"
	"os"
)

// Lock is the instance lock of a notes root (SPEC-0006).
type Lock struct{ f *os.File }

// HeldError reports that another process holds the lock.
type HeldError struct {
	Root string
	PID  int
}

func (e *HeldError) Error() string { return "not implemented" }

// AcquireLock takes the instance lock of root.
func AcquireLock(root string, log *slog.Logger) (*Lock, error) {
	return nil, errors.New("not implemented")
}

// Close releases the lock.
func (l *Lock) Close() error { return nil }
