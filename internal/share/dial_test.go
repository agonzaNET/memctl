package share

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

// TestDial_SPEC0009_R5_answerWaitEndsAtTheDeadline: an owner that accepts
// but never answers keeps a forwarder no longer than its deadline.
func TestDial_SPEC0009_R5_answerWaitEndsAtTheDeadline(t *testing.T) {
	path := filepath.Join(shortDir(t), SocketName)
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // accepted, never answered
		}
	}()
	start := time.Now()
	conn, _, err := Dial(path, NewHello("v", 1, "a <a@b>", ""), start.Add(300*time.Millisecond))
	if err == nil || conn != nil {
		t.Fatalf("got a connection from an owner that never answered")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Dial waited %v, past its deadline of 300ms", d)
	}
}
