package share

import (
	"net"
	"time"
)

// Dial connects to the owner's socket at path and sends the hello
// (SPEC-0009 R6), waiting for the answer no later than deadline (R5). On an ok answer it returns the connection; on a refusal it
// returns a nil connection and the answer, with a nil error. An error means
// that no answer came: no socket, a connection refused or closed before the
// answer (R5).
func Dial(path string, h Hello, deadline time.Time) (*net.UnixConn, Answer, error) {
	c, err := net.DialTimeout("unix", path, time.Until(deadline))
	if err != nil {
		return nil, Answer{}, err
	}
	conn := c.(*net.UnixConn)
	if err := WriteLine(conn, h); err != nil {
		conn.Close()
		return nil, Answer{}, err
	}
	conn.SetDeadline(deadline)
	a, err := ReadAnswer(conn)
	if err != nil {
		conn.Close()
		return nil, Answer{}, err
	}
	if !a.OK {
		conn.Close()
		return nil, a, nil
	}
	conn.SetDeadline(time.Time{})
	return conn, a, nil
}
