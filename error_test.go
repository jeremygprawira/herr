// Package herr_test holds the BLACK-BOX tests for herr.
//
// We deliberately use the external `herr_test` package (not `herr`) so every test
// is forced to go through the exported, public API — exactly what a real consumer
// sees. This keeps the tests coupled to BEHAVIOR, not to internal structure, so
// they survive refactors of the unexported internals.
package herr_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jeremygprawira/herr"
)

// TestNew_CarriesCode is the tracer bullet: it proves the most basic promise of the
// library end-to-end — that you can construct an error from a Code, read that Code
// back, and that the value satisfies Go's standard `error` interface with a
// developer-readable string that includes the code (useful in logs).
func TestNew_CarriesCode(t *testing.T) {
	err := herr.New("NOT_FOUND")

	// It must be a real `error` so it flows through all existing Go plumbing.
	var asError error = err
	if asError == nil {
		t.Fatal("herr.New returned something that is not a non-nil error")
	}

	// The stable, machine-readable Code is the heart of the library.
	if got := err.Code(); got != "NOT_FOUND" {
		t.Errorf("Code() = %q, want %q", got, "NOT_FOUND")
	}

	// Error() is the DEVELOPER-facing string (for logs/debugging). It should at
	// least surface the code so a log line is identifiable.
	if !strings.Contains(err.Error(), "NOT_FOUND") {
		t.Errorf("Error() = %q, want it to contain the code %q", err.Error(), "NOT_FOUND")
	}

	// Sanity: a fresh error wraps nothing yet.
	if errors.Unwrap(err) != nil {
		t.Errorf("a freshly constructed error should not wrap a cause, got %v", errors.Unwrap(err))
	}
}
