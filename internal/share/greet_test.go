package share

import (
	"net"
	"strings"
	"testing"
	"time"
)

var testOwner = Owner{PID: 4242, Version: "0.3.0", GitAuthor: "memctl <memctl@localhost>"}

// greet runs Greet on one end of a pipe after the client end wrote send, and
// returns Greet's error and the answer the client read.
func greet(t *testing.T, send string) (Answer, bool, error) {
	t.Helper()
	srv, cli := net.Pipe()
	errc := make(chan error, 1)
	go func() {
		_, err := Greet(srv, testOwner)
		errc <- err
	}()
	if send != "" {
		go cli.Write([]byte(send))
	}
	cli.SetReadDeadline(time.Now().Add(5 * time.Second))
	a, rerr := ReadAnswer(cli)
	if rerr != nil {
		t.Fatalf("no answer: %v", rerr)
	}
	err := <-errc
	// After a refusal the owner has closed the connection.
	_, cerr := cli.Read(make([]byte, 1))
	return a, cerr != nil, err
}

func TestGreet_SPEC0009_R7_refusals(t *testing.T) {
	old := HelloTimeout
	HelloTimeout = 200 * time.Millisecond
	t.Cleanup(func() { HelloTimeout = old })

	for _, tc := range []struct{ name, send, want string }{
		{"malformed", "hello\n", "not a memctl handshake"},
		{"other protocol", `{"memctl":"forward","protocol":2,"version":"v","pid":1,"git_author":"memctl <memctl@localhost>"}` + "\n", "protocol 2 is not supported"},
		{"silence", "", "no handshake within 200ms"},
		{"other author", `{"memctl":"forward","protocol":1,"version":"v","pid":1,"git_author":"Ann <ann@example.org>"}` + "\n",
			"git author differs: this process has Ann <ann@example.org>; memctl (pid 4242) serving the root has memctl <memctl@localhost>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, closed, err := greet(t, tc.send)
			if err == nil || a.OK || !strings.Contains(a.Error, tc.want) || !closed {
				t.Fatalf("err %v answer %+v closed %v; want a refusal containing %q", err, a, closed, tc.want)
			}
		})
	}
}

func TestGreet_SPEC0009_R6_okAnswer(t *testing.T) {
	srv, cli := net.Pipe()
	got := make(chan Hello, 1)
	go func() {
		h, err := Greet(srv, testOwner)
		if err != nil {
			t.Error(err)
		}
		got <- h
	}()
	go WriteLine(cli, NewHello("0.2.9", 7, "memctl <memctl@localhost>", ""))
	a, err := ReadAnswer(cli)
	if err != nil || a != (Answer{OK: true, PID: 4242, Version: "0.3.0"}) {
		t.Fatalf("answer %+v %v", a, err)
	}
	if h := <-got; h.Version != "0.2.9" || h.PID != 7 {
		t.Fatalf("hello %+v", h)
	}
}
