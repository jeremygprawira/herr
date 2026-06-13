package herr_test

import (
	"strings"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestWithStack_CapturesForServerFault proves WithStack records a stack trace for a
// server-fault kind, surfaced on the INTERNAL log Record (never on the wire). The trace
// names the capturing call site so an operator can see where the failure originated.
func TestWithStack_CapturesForServerFault(t *testing.T) {
	e := herr.New("BOOM").Kind(herr.KindInternal).WithStack()

	rec := herr.LogRecord(e)
	if rec.Stack == "" {
		t.Fatal("Stack is empty; want a captured trace for a server-fault kind")
	}
	if !strings.Contains(rec.Stack, "TestWithStack_CapturesForServerFault") {
		t.Errorf("Stack = %q, want it to name the capturing function", rec.Stack)
	}
}

// TestWithStack_SkipsClientErrors proves the H5 conditional: WithStack is a deliberate
// no-op for client-error (4xx) kinds, so call sites can use it freely without flooding logs
// with traces of ordinary, expected errors like 404s.
func TestWithStack_SkipsClientErrors(t *testing.T) {
	for _, k := range []herr.Kind{
		herr.KindInvalid, herr.KindUnauthorized, herr.KindForbidden,
		herr.KindNotFound, herr.KindConflict, herr.KindRateLimited,
	} {
		e := herr.New("X").Kind(k).WithStack()
		if rec := herr.LogRecord(e); rec.Stack != "" {
			t.Errorf("kind %v: Stack = %q, want empty (no capture for client errors)", k, rec.Stack)
		}
	}
}
