package share

import (
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"syscall"
	"time"
)

// SocketName is the file name of the owner's socket in the root's .memory
// folder (SPEC-0009 R1).
const SocketName = "memctl.sock"

// ErrPathTooLong reports a socket path longer than the platform allows
// (SPEC-0009 R2).
var ErrPathTooLong = errors.New("the socket path is longer than this platform allows")

// HelloTimeout is how long the owner waits for a forwarder's first line
// (SPEC-0009 R7). Tests shorten it.
var HelloTimeout = 5 * time.Second

// MaxPathLen is the longest socket path the platform accepts, in bytes.
func MaxPathLen() int { return len(syscall.RawSockaddrUnix{}.Path) - 1 }

// Listener is the owner's socket. Close removes the socket file.
type Listener struct{ *net.UnixListener }

// Listen removes any file at path, listens on a Unix socket there and sets its
// mode to 0600 (SPEC-0009 R1). A path that is too long returns ErrPathTooLong
// without touching the file system (R2).
func Listen(path string) (*Listener, error) {
	if len(path) > MaxPathLen() {
		return nil, ErrPathTooLong
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	return &Listener{ln}, nil
}

// Owner names the owner in the handshake.
type Owner struct {
	PID       int
	Version   string
	GitAuthor string
}

// Greet runs the owner's side of the handshake on a new connection
// (SPEC-0009 R6, R7): it reads the forwarder's hello within HelloTimeout,
// refuses a bad line or another git author with an ok-false answer and closes
// the connection, or answers ok and returns the hello.
func Greet(conn net.Conn, o Owner) (Hello, error) {
	conn.SetReadDeadline(time.Now().Add(HelloTimeout))
	tr := &trackReader{r: conn}
	h, err := ReadHello(tr)
	if err == nil {
		err = CheckAuthor(h.GitAuthor, o.GitAuthor, o.PID)
	}
	if err != nil {
		if errors.Is(tr.err, os.ErrDeadlineExceeded) {
			err = errors.New("no handshake within " + HelloTimeout.String())
		}
		conn.SetWriteDeadline(time.Now().Add(time.Second))
		WriteLine(conn, Answer{Error: err.Error()})
		conn.Close()
		return Hello{}, err
	}
	conn.SetReadDeadline(time.Time{})
	if err := WriteLine(conn, Answer{OK: true, PID: o.PID, Version: o.Version}); err != nil {
		conn.Close()
		return Hello{}, err
	}
	return h, nil
}

// trackReader remembers the last read error.
type trackReader struct {
	r   io.Reader
	err error
}

func (t *trackReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if err != nil {
		t.err = err
	}
	return n, err
}
