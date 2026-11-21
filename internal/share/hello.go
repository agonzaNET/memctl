package share

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// Protocol is the handshake protocol this memctl speaks (SPEC-0009 R6).
const Protocol = 1

// MaxLine is the longest handshake line that is read, newline included.
const MaxLine = 4096

const magic = "forward"

// Hello is the forwarder's first line (SPEC-0009 R6).
type Hello struct {
	Memctl    string `json:"memctl"`
	Protocol  int    `json:"protocol"`
	Version   string `json:"version"`
	PID       int    `json:"pid"`
	GitAuthor string `json:"git_author"`
}

// Answer is the owner's one-line answer to a Hello (SPEC-0009 R6).
type Answer struct {
	OK      bool   `json:"ok"`
	PID     int    `json:"pid,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewHello returns the hello of a forwarder with the given version, PID and
// git author.
func NewHello(version string, pid int, gitAuthor string) Hello {
	return Hello{Memctl: magic, Protocol: Protocol, Version: version, PID: pid, GitAuthor: gitAuthor}
}

// errNotHandshake is the refusal of a first line that is not a hello.
var errNotHandshake = errors.New("the first line is not a memctl handshake")

// WriteLine writes v as one JSON line, without HTML escaping, so a git author
// reads as given.
func WriteLine(w io.Writer, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// ReadHello reads and checks the forwarder's first line (SPEC-0009 R7). It
// reads byte by byte, so no byte after the line is consumed.
func ReadHello(r io.Reader) (Hello, error) {
	line, err := readLine(r)
	if err != nil {
		return Hello{}, errNotHandshake
	}
	var h Hello
	if json.Unmarshal(line, &h) != nil || h.Memctl != magic || h.Protocol == 0 ||
		h.Version == "" || h.PID <= 0 || h.GitAuthor == "" {
		return Hello{}, errNotHandshake
	}
	if h.Protocol != Protocol {
		return Hello{}, errors.New("protocol " + strconv.Itoa(h.Protocol) +
			" is not supported; this memctl speaks protocol " + strconv.Itoa(Protocol))
	}
	return h, nil
}

// ReadAnswer reads the owner's answer line.
func ReadAnswer(r io.Reader) (Answer, error) {
	line, err := readLine(r)
	if err != nil {
		return Answer{}, err
	}
	var a Answer
	if err := json.Unmarshal(line, &a); err != nil {
		return Answer{}, errors.New("the answer is not a memctl handshake line")
	}
	if !a.OK && a.Error == "" {
		return Answer{}, errors.New("the answer is not a memctl handshake line")
	}
	return a, nil
}

// CheckAuthor returns the refusal of SPEC-0009 R7 when the forwarder's git
// author differs from the owner's.
func CheckAuthor(forwarder, owner string, ownerPID int) error {
	if forwarder == owner {
		return nil
	}
	return errors.New("git author differs: this process has " + forwarder +
		"; memctl (pid " + strconv.Itoa(ownerPID) + ") serving the root has " + owner)
}

// readLine reads one line of at most MaxLine bytes, newline included, and
// returns it without the newline.
func readLine(r io.Reader) ([]byte, error) {
	var line []byte
	b := make([]byte, 1)
	for len(line) < MaxLine {
		n, err := r.Read(b)
		if n == 1 {
			if b[0] == '\n' {
				return line, nil
			}
			line = append(line, b[0])
			continue
		}
		if err != nil {
			if err == io.EOF {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
	}
	return nil, errors.New("handshake line longer than " + strconv.Itoa(MaxLine) + " bytes")
}
