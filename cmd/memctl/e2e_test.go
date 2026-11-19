package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tee copies everything the server writes to stdout into a buffer, so the
// test can check that stdout holds only protocol messages.
type tee struct {
	r   io.ReadCloser
	mu  sync.Mutex
	buf bytes.Buffer
}

func (t *tee) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	t.mu.Lock()
	t.buf.Write(p[:n])
	t.mu.Unlock()
	return n, err
}

func (t *tee) Close() error { return t.r.Close() }

func (t *tee) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

// TestE2E_SPEC0004_R3_R4_R5_protocolOverPipes spawns memctl mcp and talks to
// it with the SDK client over its stdin and stdout.
func TestE2E_SPEC0004_R3_R4_R5_protocolOverPipes(t *testing.T) {
	root := notesRoot(t)
	cmd := exec.Command(binVersioned, "mcp", "--root", root)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	var stderr syncBuffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	out := &tee{r: stdout}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil)
	cs, err := client.Connect(ctx, &mcp.IOTransport{Reader: out, Writer: stdin}, nil)
	if err != nil {
		t.Fatalf("connect: %v; stderr %s", err, stderr.String())
	}
	init := cs.InitializeResult()
	if init.ServerInfo.Name != "memctl" || init.ServerInfo.Version != "9.8.7-test" || init.Instructions == "" {
		t.Fatalf("initialize %+v", init)
	}
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) == 0 {
		t.Fatalf("tools %v %v", tools, err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "note_read", Arguments: map[string]any{"ref": "a"}})
	if err != nil || res.IsError {
		t.Fatalf("note_read: %v %+v", err, res)
	}
	cs.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("exit: %v; stderr %s", err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("did not exit after stdin closed")
	}

	sc := bufio.NewScanner(strings.NewReader(out.String()))
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil || m["jsonrpc"] != "2.0" {
			t.Fatalf("stdout line is not a protocol message: %q", sc.Text())
		}
	}
	lines := jsonLines(t, stderr.String())
	if byMsg(lines, "started") == nil || byMsg(lines, "stopped") == nil || byMsg(lines, "tool call") == nil {
		t.Fatalf("stderr lines %v", lines)
	}
}
