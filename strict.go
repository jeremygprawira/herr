package herr

import "sync/atomic"

// StrictMode toggles a process-wide development aid. herr is built around "magic is a
// fallback, never a trap" — soft failures (like a {param} you forgot to supply) quietly do
// the safe thing in production rather than panic. StrictMode flips those soft failures into
// LOUD, VISIBLE ones during development and tests, so mistakes surface early instead of
// shipping as subtly wrong output.
//
// Currently it governs template substitution (H3): an unfilled {name} placeholder collapses
// to empty in production, but is left visible (as the literal "{name}") in StrictMode. It
// never panics in either mode.
//
// The flag lives behind an atomic so toggling it (typically once in TestMain or an init for
// a dev build) cannot data-race with concurrent renders.
func StrictMode(on bool) {
	strict.Store(on)
}

// strict holds the StrictMode flag. The zero value (false) is production behavior, so the
// safe default needs no initialization.
var strict atomic.Bool

// strictModeOn reports whether StrictMode is currently enabled. Read lock-free on the render
// path.
func strictModeOn() bool {
	return strict.Load()
}
