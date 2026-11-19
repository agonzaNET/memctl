package note

import "errors"

// NewNote is the input of Write.
type NewNote struct {
	Title, Folder, Body, Type string
	Tags                      []string
}

// Written is the result of Write.
type Written struct {
	Path, Title, SHA256 string
	Bytes               int
	SameTitle           []string
}

// ExistsError reports a note at the target path.
type ExistsError struct{ Path string }

func (e *ExistsError) Error() string { return "not implemented" }

// ArgError is an argument that breaks a rule.
type ArgError struct{ Arg, Reason string }

func (e *ArgError) Error() string { return "not implemented" }

// ErrFrontmatterInBody is returned for a body that starts with frontmatter.
var ErrFrontmatterInBody = errors.New("not implemented")

// FileName returns the file name of a note title.
func FileName(title string) (string, error) { return "", errTODO }

// Write creates a note.
func (s *Service) Write(n NewNote) (Written, error) { return Written{}, errTODO }

var errTODO = errors.New("not implemented")
