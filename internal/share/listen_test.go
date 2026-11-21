package share

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListen_SPEC0009_R1_replacesStaleFileAndSetsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), SocketName)
	if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	ln, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSocket == 0 || fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want a socket with 0600", fi.Mode())
	}
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	ln.Close()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("socket file after Close: %v", err)
	}
}

func TestListen_SPEC0009_R2_tooLongPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, strings.Repeat("s", MaxPathLen()))
	if _, err := Listen(path); err != ErrPathTooLong {
		t.Fatalf("got %v, want ErrPathTooLong", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("a too-long path touched the file system: %v", err)
	}
}
