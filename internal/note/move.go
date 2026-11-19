package note

// MoveOp is the input of Move.
type MoveOp struct{ NewPath, NewTitle string }

// Moved is the result of Move.
type Moved struct {
	OldPath, Path, Title, SHA256 string
	Bytes, Linkers               int
	Same                         bool
}

// Move moves or renames a note.
func (s *Service) Move(resolve Resolver, op MoveOp) (Moved, error) { return Moved{}, ErrConflict }
