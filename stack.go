package herr

import (
	"runtime"
	"strconv"
	"strings"
)

// Stacks are an INTERNAL, operator-only aid: they show WHERE an unexpected failure
// originated. They are valuable for server faults (a 500 you didn't see coming) but pure
// noise for expected client errors (a 404 is normal control flow), and they are not free —
// capturing and formatting frames costs CPU and memory. So capture is CONDITIONAL (H5):
// WithStack records a trace only for server-fault kinds and is a no-op otherwise.

// serverFaultKinds is the set of Kinds for which a stack trace is worth capturing: the
// 5xx, "something broke on our side" classifications. Client-error kinds (4xx) are
// deliberately absent — they are expected outcomes, not bugs to be traced.
var serverFaultKinds = map[Kind]bool{
	KindInternal:    true, // 500 — unexpected fault
	KindUnavailable: true, // 503 — dependency/service down
	KindTimeout:     true, // 504 — deadline exceeded
}

// WithStack captures the current call stack and attaches it to the error — but ONLY when
// the error's Kind is a server fault (see serverFaultKinds). For any other Kind it is a
// deliberate no-op, so sprinkling WithStack at call sites never floods logs with traces of
// ordinary 4xx errors. The captured trace lives on the INTERNAL surface (Record.Stack) and
// never crosses the wire. Returns the receiver for chaining.
func (e *Error) WithStack() *Error {
	if e == nil {
		return nil
	}
	if !serverFaultKinds[e.kind] {
		return e // conditional capture (H5): skip client-error kinds
	}
	e.stack = captureStack()
	return e
}

// captureStack records the call stack of WithStack's caller and formats it as a readable,
// log-friendly string ("function\n\tfile:line" per frame). It skips its own frames so the
// trace starts at the code that actually called WithStack.
func captureStack() string {
	// Skip three frames: runtime.Callers, captureStack, and WithStack — so the first
	// frame is the caller of WithStack.
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	if n == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:n])

	var b strings.Builder
	for {
		f, more := frames.Next()
		b.WriteString(f.Function)
		b.WriteString("\n\t")
		b.WriteString(f.File)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(f.Line))
		b.WriteByte('\n')
		if !more {
			break
		}
	}
	return b.String()
}
