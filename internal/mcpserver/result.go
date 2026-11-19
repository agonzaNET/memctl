package mcpserver

// Code is a tool error code from the fixed list (SPEC-0004, "Error
// handling"). Tools can only emit the constants below.
type Code string

const (
	CodeInvalidArgument Code = "invalid_argument"
	CodeNotFound        Code = "not_found"
	CodeAmbiguous       Code = "ambiguous"
	CodeInternalError   Code = "internal_error"
)

// Codes lists every code a tool may emit.
var Codes = []Code{CodeInvalidArgument, CodeNotFound, CodeAmbiguous, CodeInternalError}
