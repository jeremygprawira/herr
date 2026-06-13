package herr_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jeremygprawira/herr"
)

// TestMultiError_JoinPromotesFieldErrors proves herr is multi-error-agnostic: when a parent
// wraps a stdlib errors.Join aggregate, any *herr.Error children are promoted into the
// public errors[] using only their public parts — so a handler can keep using errors.Join
// (or any aggregator) and still get a clean, per-field response.
func TestMultiError_JoinPromotesFieldErrors(t *testing.T) {
	joined := errors.Join(
		herr.FieldError("email", "INVALID_EMAIL", "Enter a valid email address."),
		herr.FieldError("age", "TOO_YOUNG", "Must be 18 or older."),
	)
	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).Wrap(joined)

	entries := fieldErrorEntries(t, e, "en")
	if len(entries) != 2 {
		t.Fatalf("errors len = %d, want 2", len(entries))
	}
	if entries[0]["field"] != "email" || entries[0]["code"] != "INVALID_EMAIL" {
		t.Errorf("entry[0] = %v, want email/INVALID_EMAIL", entries[0])
	}
	if entries[1]["field"] != "age" || entries[1]["code"] != "TOO_YOUNG" {
		t.Errorf("entry[1] = %v, want age/TOO_YOUNG", entries[1])
	}
}

// fakeMultierror mimics hashicorp/go-multierror's shape: it exposes its children via a
// WrappedErrors() []error method (NOT the stdlib Unwrap() []error). herr must recognize it
// structurally, without importing the library — proving the "multi-error-agnostic" claim.
type fakeMultierror struct{ errs []error }

func (m *fakeMultierror) Error() string          { return "multiple errors" }
func (m *fakeMultierror) WrappedErrors() []error { return m.errs }

// TestMultiError_WrappedErrorsShapePromotes proves the go-multierror shape works too: a
// parent wrapping a value that exposes WrappedErrors() has its *herr.Error children promoted
// into errors[] exactly like errors.Join.
func TestMultiError_WrappedErrorsShapePromotes(t *testing.T) {
	agg := &fakeMultierror{errs: []error{
		herr.FieldError("password", "TOO_WEAK", "Use at least 12 characters."),
	}}
	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).Wrap(agg)

	entries := fieldErrorEntries(t, e, "en")
	if len(entries) != 1 {
		t.Fatalf("errors len = %d, want 1", len(entries))
	}
	if entries[0]["field"] != "password" || entries[0]["code"] != "TOO_WEAK" {
		t.Errorf("entry = %v, want password/TOO_WEAK", entries[0])
	}
}

// TestMultiError_NonHerrChildNeverLeaks is the C2 guard for promotion: when an aggregate
// mixes a herr field error with an arbitrary NON-herr error, only the herr child is
// promoted to errors[]. The arbitrary error's string (which may carry raw internals) must
// never appear in the wire body — but it stays visible to logs via the wrapped cause.
func TestMultiError_NonHerrChildNeverLeaks(t *testing.T) {
	const secret = "raw db error: password=hunter2"

	joined := errors.Join(
		herr.FieldError("email", "INVALID_EMAIL", "Enter a valid email address."),
		errors.New(secret), // arbitrary non-herr error
	)
	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).Wrap(joined)

	entries := fieldErrorEntries(t, e, "en")
	if len(entries) != 1 {
		t.Fatalf("errors len = %d, want 1 (non-herr child must not be promoted)", len(entries))
	}

	raw, _ := json.Marshal(e.Body("en"))
	if strings.Contains(string(raw), "hunter2") {
		t.Errorf("LEAK: non-herr child surfaced in wire body: %s", raw)
	}

	// The non-herr child is not lost — it remains available to operators via the cause.
	if c := herr.LogRecord(e).Cause; c == nil || !strings.Contains(c.Error(), secret) {
		t.Error("non-herr child should remain visible to logs via the wrapped cause")
	}
}
