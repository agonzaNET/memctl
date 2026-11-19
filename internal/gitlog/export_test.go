package gitlog

import "time"

// SetIdle installs a hook that runs each time the loop is about to wait.
func SetIdle(c *Committer, f func()) { c.onIdle = f }

// Message is the commit message of a batch.
var Message = message

// SetTimeout changes the timeout of the committer's git calls.
func SetTimeout(c *Committer, d time.Duration) { c.timeout = d }
