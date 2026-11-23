package main

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxScope is the longest scope, in Unicode code points (SPEC-0011 R3).
const maxScope = 200

// scopeRule is the rule of --scope, named in its usage error (SPEC-0011 R3).
const scopeRule = "must be one line of at most 200 characters of valid UTF-8, with no control characters"

// parseScope returns the scope of memctl mcp: the --scope value when the flag
// was given, even empty, and MEMCTL_SCOPE otherwise, with surrounding white
// space removed. An empty result means no scope (SPEC-0011 R1, R2). A value
// that breaks the rule gives an error naming its source, not the value (R3).
func parseScope(flagValue string, flagSet bool, getenv func(string) string) (string, error) {
	source, v := "--scope", flagValue
	if !flagSet {
		source, v = "MEMCTL_SCOPE", getenv("MEMCTL_SCOPE")
	}
	if !utf8.ValidString(v) {
		return "", errors.New(source + " " + scopeRule)
	}
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) > maxScope || strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return "", errors.New(source + " " + scopeRule)
	}
	return v, nil
}
