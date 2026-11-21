package mcpserver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestServer_SPEC0009_R12_sessionsShareOneServer: the stdio session and a
// forwarded connection are two sessions of one server over one note service;
// a note written through one is read through the other.
func TestServer_SPEC0009_R12_sessionsShareOneServer(t *testing.T) {
	e := start(t, map[string]string{"a.md": "---\ntitle: A\n---\nbody\n"})

	srvSide, cliSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.srv.ServeConn(ctx, srvSide, "forward-1") }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("forwarded session did not stop")
		}
	})
	cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	fwd, err := mcp.NewClient(&mcp.Implementation{Name: "fwd", Version: "0"}, nil).
		Connect(cctx, &mcp.IOTransport{Reader: cliSide, Writer: cliSide}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()

	res, err := fwd.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_write",
		Arguments: map[string]any{"title": "From the forwarder", "body": "shared"}})
	if err != nil || res.IsError {
		t.Fatalf("note_write through the forwarded session: %v %+v", err, res)
	}
	if got := e.call(t, "note_read", map[string]any{"ref": "From the forwarder"}); got.IsError {
		t.Fatalf("note_read through the stdio session: %s", text(t, got))
	}
}

// TestServer_SPEC0009_R12_forwardedSessionEndsAlone: the end of a forwarded
// session leaves the stdio session working.
func TestServer_SPEC0009_R12_forwardedSessionEndsAlone(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	srvSide, cliSide := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- e.srv.ServeConn(context.Background(), srvSide, "forward-1") }()
	cliSide.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ServeConn did not return after its connection closed")
	}
	if got := e.call(t, "note_read", map[string]any{"ref": "a"}); got.IsError {
		t.Fatalf("stdio session after the forwarded one ended: %s", text(t, got))
	}
}

// TestServer_SPEC0009_R15_forwardedCallFinishes: on shutdown, a forwarded
// session returns only after its tool call in progress has finished.
func TestServer_SPEC0009_R15_forwardedCallFinishes(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	release := make(chan struct{})
	entered := make(chan struct{})
	var once bool
	e.srv.SetBeforeCall(func() {
		if !once {
			once = true
			close(entered)
			<-release
		}
	})
	srvSide, cliSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.srv.ServeConn(ctx, srvSide, "forward-1") }()
	cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	fwd, err := mcp.NewClient(&mcp.Implementation{Name: "fwd", Version: "0"}, nil).
		Connect(cctx, &mcp.IOTransport{Reader: cliSide, Writer: cliSide}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()
	go fwd.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_read", Arguments: map[string]any{"ref": "a"}})
	<-entered
	cancel()
	select {
	case <-done:
		t.Fatal("ServeConn returned while its call was in progress")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ServeConn did not return after the call finished")
	}
}

// TestLog_SPEC0009_R13_sessionField: each tool call line names its session.
func TestLog_SPEC0009_R13_sessionField(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	srvSide, cliSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.srv.ServeConn(ctx, srvSide, "forward-7") }()
	t.Cleanup(func() { cancel(); <-done })
	cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	fwd, err := mcp.NewClient(&mcp.Implementation{Name: "fwd", Version: "0"}, nil).
		Connect(cctx, &mcp.IOTransport{Reader: cliSide, Writer: cliSide}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()
	e.call(t, "note_read", map[string]any{"ref": "a"})
	if _, err := fwd.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_read", Arguments: map[string]any{"ref": "a"}}); err != nil {
		t.Fatal(err)
	}
	lines := e.logLines(t, "tool call")
	if len(lines) != 2 || lines[0]["session"] != "stdio" || lines[1]["session"] != "forward-7" {
		t.Fatalf("tool call lines %v, want sessions stdio and forward-7", lines)
	}
}

// TestServer_SPEC0009_R12_concurrentSessions runs calls from two sessions at
// once, for the race detector.
func TestServer_SPEC0009_R12_concurrentSessions(t *testing.T) {
	e := start(t, map[string]string{"a.md": "---\ntitle: A\n---\nfirst\n"})
	srvSide, cliSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.srv.ServeConn(ctx, srvSide, "forward-1") }()
	t.Cleanup(func() { cancel(); <-done })
	cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ccancel()
	fwd, err := mcp.NewClient(&mcp.Implementation{Name: "fwd", Version: "0"}, nil).
		Connect(cctx, &mcp.IOTransport{Reader: cliSide, Writer: cliSide}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()
	finished := make(chan struct{}, 2)
	for _, cs := range []*mcp.ClientSession{e.cs, fwd} {
		go func() {
			for i := 0; i < 20; i++ {
				cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "search", Arguments: map[string]any{"query": "first"}})
				cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_write", Arguments: map[string]any{"title": "N", "body": "x", "overwrite": true}})
			}
			finished <- struct{}{}
		}()
	}
	<-finished
	<-finished
}
