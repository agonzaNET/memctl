// Command memctl serves a folder of Markdown notes to MCP clients over stdio
// (memctl mcp) and reports problems in it (memctl check).
package main

import (
	"io"
	"os"
)

// version is set at build time with -ldflags "-X main.version=<value>".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes one command line and returns the exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return 0
}
