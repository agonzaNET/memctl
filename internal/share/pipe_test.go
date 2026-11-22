package share

import (
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

// unixPair is a connected pair of Unix stream sockets.
func unixPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(shortDir(t), "s"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	acc := make(chan *net.UnixConn, 1)
	go func() {
		c, err := l.AcceptUnix()
		if err != nil {
			acc <- nil
			return
		}
		acc <- c
	}()
	c, err := net.DialUnix("unix", nil, l.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	o := <-acc
	if o == nil {
		t.Fatal("accept failed")
	}
	return c, o
}

// TestPipe_SPEC0009_R10_R11_stdinEOFIsNoOwnerLoss: when the owner closes the
// connection in answer to stdin EOF, Pipe never reports the owner as lost.
func TestPipe_SPEC0009_R10_R11_stdinEOFIsNoOwnerLoss(t *testing.T) {
	for i := 0; i < 500; i++ {
		c, o := unixPair(t)
		go func() {
			io.Copy(io.Discard, o) // the owner reads to EOF, then ends at once
			o.Close()
		}()
		if Pipe(strings.NewReader("{}\n"), io.Discard, c) {
			t.Fatalf("run %d: owner reported lost after stdin EOF", i)
		}
		c.Close()
	}
}

// TestPipe_SPEC0009_R11_ownerLoss: the owner closing while stdin is open is
// reported.
func TestPipe_SPEC0009_R11_ownerLoss(t *testing.T) {
	c, o := unixPair(t)
	o.Close()
	r, w := io.Pipe()
	defer w.Close()
	if !Pipe(r, io.Discard, c) {
		t.Fatal("owner loss not reported")
	}
}
