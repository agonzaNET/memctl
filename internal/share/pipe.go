package share

import (
	"io"
	"net"
)

// Pipe copies stdin to conn and conn to stdout, byte for byte, in both
// directions at once (SPEC-0009 R9). On stdin EOF it closes the write half of
// conn and keeps copying to stdout until the owner closes conn (R10). It
// returns ownerLost true when conn ended while stdin was still open (R11).
func Pipe(stdin io.Reader, stdout io.Writer, conn *net.UnixConn) (ownerLost bool) {
	stdinDone := make(chan struct{})
	go func() {
		io.Copy(conn, stdin)
		// Marked before CloseWrite: the owner may close conn as soon as it
		// sees EOF, and that must not read as a lost owner (R11).
		close(stdinDone)
		conn.CloseWrite()
	}()
	io.Copy(stdout, conn)
	select {
	case <-stdinDone:
		return false
	default:
		return true
	}
}
