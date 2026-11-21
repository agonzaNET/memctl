package share

import (
	"bytes"
	"strings"
	"testing"
)

func TestHello_SPEC0009_R6_oneLineEachWay(t *testing.T) {
	var buf bytes.Buffer
	h := NewHello("0.3.0", 4711, "memctl <memctl@localhost>")
	if err := WriteLine(&buf, h); err != nil {
		t.Fatal(err)
	}
	want := `{"memctl":"forward","protocol":1,"version":"0.3.0","pid":4711,"git_author":"memctl <memctl@localhost>"}` + "\n"
	if buf.String() != want {
		t.Fatalf("hello line\n got %q\nwant %q", buf.String(), want)
	}
	got, err := ReadHello(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got != h {
		t.Fatalf("round trip: got %+v, want %+v", got, h)
	}

	for _, tc := range []struct {
		a    Answer
		line string
	}{
		{Answer{OK: true, PID: 4242, Version: "0.3.0"}, `{"ok":true,"pid":4242,"version":"0.3.0"}`},
		{Answer{Error: "git author differs"}, `{"ok":false,"error":"git author differs"}`},
	} {
		buf.Reset()
		if err := WriteLine(&buf, tc.a); err != nil {
			t.Fatal(err)
		}
		if buf.String() != tc.line+"\n" {
			t.Fatalf("answer line\n got %q\nwant %q", buf.String(), tc.line+"\n")
		}
		got, err := ReadAnswer(&buf)
		if err != nil || got != tc.a {
			t.Fatalf("answer round trip: got %+v, %v; want %+v", got, err, tc.a)
		}
	}
}

func TestReadHello_SPEC0009_R6_readsNoByteAfterTheLine(t *testing.T) {
	r := strings.NewReader(`{"memctl":"forward","protocol":1,"version":"v","pid":1,"git_author":"a <a@b>"}` + "\n" + `{"jsonrpc":"2.0"}`)
	if _, err := ReadHello(r); err != nil {
		t.Fatal(err)
	}
	rest := new(bytes.Buffer)
	rest.ReadFrom(r)
	if rest.String() != `{"jsonrpc":"2.0"}` {
		t.Fatalf("bytes after the hello: got %q", rest.String())
	}
}

func TestReadHello_SPEC0009_R7_refusesBadFirstLines(t *testing.T) {
	long := strings.Repeat("x", MaxLine+1)
	for _, tc := range []struct {
		name, in, want string
	}{
		{"not JSON", "hello\n", "not a memctl handshake"},
		{"other magic", `{"memctl":"other","protocol":1,"version":"v","pid":1,"git_author":"a <a@b>"}` + "\n", "not a memctl handshake"},
		{"missing pid", `{"memctl":"forward","protocol":1,"version":"v","git_author":"a <a@b>"}` + "\n", "not a memctl handshake"},
		{"missing author", `{"memctl":"forward","protocol":1,"version":"v","pid":1}` + "\n", "not a memctl handshake"},
		{"other protocol", `{"memctl":"forward","protocol":2,"version":"v","pid":1,"git_author":"a <a@b>"}` + "\n", "protocol 2 is not supported"},
		{"no newline", `{"memctl":"forward"`, "not a memctl handshake"},
		{"too long", long + "\n", "not a memctl handshake"},
		{"empty", "", "not a memctl handshake"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadHello(strings.NewReader(tc.in))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
	// Extra fields are ignored, so a later protocol-1 forwarder may add some.
	if _, err := ReadHello(strings.NewReader(`{"memctl":"forward","protocol":1,"version":"v","pid":1,"git_author":"a <a@b>","extra":true}` + "\n")); err != nil {
		t.Fatalf("extra field: %v", err)
	}
}

func TestAuthorCheck_SPEC0009_R7_message(t *testing.T) {
	if err := CheckAuthor("memctl <memctl@localhost>", "memctl <memctl@localhost>", 4242); err != nil {
		t.Fatalf("same author: %v", err)
	}
	err := CheckAuthor("Ann <ann@example.org>", "memctl <memctl@localhost>", 4242)
	want := "git author differs: this process has Ann <ann@example.org>; memctl (pid 4242) serving the root has memctl <memctl@localhost>"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %s", err, want)
	}
}
