// Command memctl serves a folder of Markdown notes to MCP clients over stdio
// (memctl mcp) and reports problems in it (memctl check).
package main

import (
	"fmt"
	"io"
	"os"
)

// version is set at build time with -ldflags "-X main.version=<value>".
var version = "dev"

const usageText = `usage:
  memctl mcp [--root DIR] [--log-level LEVEL]
        Serve the notes under DIR to an MCP client over stdio.
        --root       notes root (or MEMCTL_ROOT); required
        --log-level  debug, info, warn or error (or MEMCTL_LOG_LEVEL); default info
  memctl check [--root DIR]
        Report problems in the notes under DIR; exits 1 when there are errors.
  memctl version
        Print the version.
`

// Exit statuses (SPEC-0004 R1, R2).
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes one command line and returns the exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "a command is required")
	}
	switch args[0] {
	case "version":
		if len(args) > 1 {
			return usageError(stderr, "memctl version takes no arguments")
		}
		fmt.Fprintf(stdout, "memctl %s\n", version)
		return exitOK
	case "mcp":
		return runMCP(args[1:], stdin, stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	default:
		return usageError(stderr, "unknown command")
	}
}

// usageError prints msg and the usage to stderr and returns the usage status.
// msg names the flag and the rule, never the value (SPEC-0004 R1).
func usageError(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "memctl: %s\n\n%s", msg, usageText)
	return exitUsage
}
