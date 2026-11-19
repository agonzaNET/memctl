package parse

import "errors"

// FrontmatterSet sets one key.
type FrontmatterSet struct{ Key, Value string }

// ErrNoFrontmatter is returned for a note without editable frontmatter.
var ErrNoFrontmatter = errors.New("not implemented")

// EditFrontmatter edits the frontmatter text.
func EditFrontmatter(data []byte, changes ...FrontmatterSet) ([]byte, error) {
	return nil, errors.New("not implemented")
}
