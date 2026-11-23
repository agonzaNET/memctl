package share

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

const scope = "Alberto's personal notes. No appliedAI work."

func TestHello_SPEC0011_R11_scopeOnlyWhenSet(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLine(&buf, NewHello("0.4.0", 4711, "memctl <memctl@localhost>", scope)); err != nil {
		t.Fatal(err)
	}
	want := `{"memctl":"forward","protocol":1,"version":"0.4.0","pid":4711,"git_author":"memctl <memctl@localhost>","scope":"Alberto's personal notes. No appliedAI work."}` + "\n"
	if buf.String() != want {
		t.Fatalf("hello line\n got %q\nwant %q", buf.String(), want)
	}
	h, err := ReadHello(&buf)
	if err != nil || h.Scope != scope {
		t.Fatalf("round trip: %+v %v", h, err)
	}
	buf.Reset()
	WriteLine(&buf, NewHello("0.4.0", 4711, "memctl <memctl@localhost>", ""))
	if strings.Contains(buf.String(), "scope") {
		t.Fatalf("hello without a scope carries one: %q", buf.String())
	}

	for _, tc := range []struct {
		a    Answer
		line string
	}{
		{Answer{OK: true, PID: 4242, Version: "0.4.0", Scope: scope}, `{"ok":true,"pid":4242,"version":"0.4.0","scope":"Alberto's personal notes. No appliedAI work."}`},
		{Answer{OK: true, PID: 4242, Version: "0.4.0"}, `{"ok":true,"pid":4242,"version":"0.4.0"}`},
	} {
		buf.Reset()
		WriteLine(&buf, tc.a)
		if buf.String() != tc.line+"\n" {
			t.Fatalf("answer line\n got %q\nwant %q", buf.String(), tc.line+"\n")
		}
		got, err := ReadAnswer(&buf)
		if err != nil || got != tc.a {
			t.Fatalf("answer round trip: %+v %v; want %+v", got, err, tc.a)
		}
	}
}

func TestHello_SPEC0011_R11_oldAndUnknownFieldsAccepted(t *testing.T) {
	// A hello of a memctl without scopes, and one with a field this memctl
	// does not know: both are accepted, protocol 1, no scope.
	for _, line := range []string{
		`{"memctl":"forward","protocol":1,"version":"0.3.2","pid":7,"git_author":"a <a@b>"}`,
		`{"memctl":"forward","protocol":1,"version":"0.9.0","pid":7,"git_author":"a <a@b>","later":true}`,
	} {
		h, err := ReadHello(strings.NewReader(line + "\n"))
		if err != nil || h.Scope != "" || h.Protocol != 1 {
			t.Fatalf("%s: %+v %v", line, h, err)
		}
	}
	for _, line := range []string{`{"ok":true,"pid":1,"version":"0.3.2"}`, `{"ok":true,"pid":1,"version":"0.9.0","later":1}`} {
		a, err := ReadAnswer(strings.NewReader(line + "\n"))
		if err != nil || !a.OK || a.Scope != "" {
			t.Fatalf("%s: %+v %v", line, a, err)
		}
	}
	if Protocol != 1 {
		t.Fatalf("protocol %d, want 1", Protocol)
	}
}

func TestGreet_SPEC0011_R11_okAnswerCarriesTheOwnersScope(t *testing.T) {
	srv, cli := net.Pipe()
	o := testOwner
	o.Scope = scope
	got := make(chan Hello, 1)
	go func() {
		h, err := Greet(srv, o)
		if err != nil {
			t.Error(err)
		}
		got <- h
	}()
	go WriteLine(cli, NewHello("0.4.0", 7, "memctl <memctl@localhost>", "work notes"))
	a, err := ReadAnswer(cli)
	if err != nil || a != (Answer{OK: true, PID: 4242, Version: "0.3.0", Scope: scope}) {
		t.Fatalf("answer %+v %v", a, err)
	}
	if h := <-got; h.Scope != "work notes" {
		t.Fatalf("hello %+v", h)
	}
}
