package herr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestIs_MatchesByCode proves herr errors play nicely with the standard errors package:
// they are identified by their stable Code, the catalog handle offers an ergonomic
// .Is(err) check, and matching works even when the herr error is buried under wrapping
// from other packages.
func TestIs_MatchesByCode(t *testing.T) {
	var ErrNotFound = herr.Define(herr.Class{Code: "NOT_FOUND", Kind: herr.KindNotFound})
	var ErrConflict = herr.Define(herr.Class{Code: "CONFLICT", Kind: herr.KindConflict})

	// A herr error wrapping a low-level cause.
	e := ErrNotFound.New().Wrap(errors.New("sql: no rows in result set"))

	// Catalog .Is convenience: "is this error a NOT_FOUND?"
	if !ErrNotFound.Is(e) {
		t.Error("ErrNotFound.Is(e) = false, want true")
	}
	if ErrConflict.Is(e) {
		t.Error("ErrConflict.Is(e) = true, want false (different code)")
	}

	// Buried under foreign wrapping, the match still holds (chain walking via Unwrap).
	wrapped := fmt.Errorf("loading user: %w", e)
	if !ErrNotFound.Is(wrapped) {
		t.Error("ErrNotFound.Is(wrapped) = false, want true through the chain")
	}

	// Standard errors.As extracts the concrete *herr.Error so callers can read its code,
	// status, etc.
	var he *herr.Error
	if !errors.As(wrapped, &he) {
		t.Fatal("errors.As could not extract *herr.Error from the chain")
	}
	if he.Code() != "NOT_FOUND" {
		t.Errorf("extracted code = %q, want NOT_FOUND", he.Code())
	}

	// Standard errors.Is against a concrete herr target also matches by code.
	if !errors.Is(wrapped, ErrNotFound.New()) {
		t.Error("errors.Is(wrapped, NOT_FOUND error) = false, want true")
	}
}
