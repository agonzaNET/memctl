package note

import (
	"errors"

	"github.com/agonzaNET/memctl/internal/index"
)

// ErrConflict is returned when a file changed after the freshness check.
var ErrConflict = errors.New("note: the file changed")

// PathError is a rejected tool path.
type PathError struct{ Path, Reason string }

func (e *PathError) Error() string { return "not implemented" }
func (e *PathError) Unwrap() error { return ErrBadPath }

var errTODO = errors.New("not implemented")

func (s *Service) mutate(fn func(idx *index.Index) (Change, error)) error { return errTODO }
func (s *Service) putFile(rel string, data []byte, prevSHA string) (index.Meta, error) {
	return index.Meta{}, errTODO
}
func (s *Service) removeFile(rel, prevSHA string) error  { return errTODO }
func (s *Service) targetPath(rel string) (string, error) { return "", errTODO }
func (s *Service) stamp() string                         { return "" }
