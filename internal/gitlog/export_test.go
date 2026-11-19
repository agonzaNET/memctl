package gitlog

// SetIdle installs a hook that runs each time the loop is about to wait.
func SetIdle(c *Committer, f func()) { c.onIdle = f }

// Message is the commit message of a batch.
var Message = message
