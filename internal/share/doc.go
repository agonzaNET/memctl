// Package share lets several memctl mcp processes serve one notes root: the
// owner, which holds the instance lock, listens on a Unix socket in the
// root's .memory folder, and every later process forwards its client's bytes
// to it after a one-line handshake (SPEC-0009, ADR-0021). It uses the
// standard library only.
package share
