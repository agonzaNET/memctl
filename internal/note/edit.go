package note

import (
	"errors"

	"github.com/agonzaNET/memctl/internal/index"
)

// Resolver resolves a tool ref.
type Resolver func(idx *index.Index) (string, error)

// EditOp is the input of Edit.
type EditOp struct{ ExpectedSHA, Op, Text, Section, Find, Replace string }

// Edited is the result of Edit.
type Edited struct {
	Path, Title, SHA256 string
	Bytes               int
	Unchanged           bool
}

// ConflictError is a stale expected_sha.
type ConflictError struct{ SHA256, Content string }

func (e *ConflictError) Error() string { return "not implemented" }
func (e *ConflictError) Unwrap() error { return ErrConflict }

// SectionError is a missing heading.
type SectionError struct {
	Section  string
	Headings []string
}

func (e *SectionError) Error() string { return "not implemented" }

// NotUniqueError is a find text that occurs more than once.
type NotUniqueError struct{ Count int }

func (e *NotUniqueError) Error() string { return "not implemented" }

// ErrNoMatch is a find text that does not occur.
var ErrNoMatch = errors.New("not implemented")

// Edit changes a note.
func (s *Service) Edit(resolve Resolver, op EditOp) (Edited, error) { return Edited{}, errTODO }

var errTODO = errors.New("not implemented")
