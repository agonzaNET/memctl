package gitlog

import (
	"bytes"
	"context"
	"strings"
)

// Scan queues as external changes the notes that git reports as changed,
// added, deleted or renamed in the notes root, so changes made while memctl
// was not running are committed too (SPEC-0008 R15). isNote says whether a
// path relative to the root is one the walk treats as a note; a deleted
// path counts when it ends in ".md". Scan does nothing when git is off.
func (c *Committer) Scan(ctx context.Context, isNote func(rel string) bool) error {
	if !c.enabled {
		return nil
	}
	prefix, err := runGit(ctx, c.root, c.timeout, nil, nil, "rev-parse", "--show-prefix")
	if err != nil {
		return err
	}
	pre := strings.TrimSpace(string(prefix))
	out, err := runGit(ctx, c.root, c.timeout, nil, nil, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		return err
	}
	var paths []string
	add := func(repoPath string, deleted bool) {
		rel, ok := strings.CutPrefix(repoPath, pre)
		if !ok || rel == "" {
			return
		}
		if (deleted && strings.HasSuffix(rel, ".md")) || (!deleted && isNote(rel)) {
			paths = append(paths, rel)
		}
	}
	fields := bytes.Split(out, []byte{0})
	for i := 0; i < len(fields); i++ {
		f := string(fields[i])
		if len(f) < 4 {
			continue
		}
		xy, p := f[:2], f[3:]
		switch {
		case xy[0] == 'R' || xy[0] == 'C':
			// "R  new\0old": the new path, then the old one.
			add(p, false)
			if i+1 < len(fields) {
				i++
				if xy[0] == 'R' {
					add(string(fields[i]), true)
				}
			}
		case xy[0] == 'D' || xy[1] == 'D':
			add(p, true)
		default:
			add(p, false)
		}
	}
	c.External(paths)
	return nil
}
