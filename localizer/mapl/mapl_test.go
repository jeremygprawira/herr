// Package mapl_test black-box tests the map-backed Localizer. It uses only the
// public API of package mapl and the real herr core — never package internals —
// which mirrors how a real consumer would wire a translation table into herr.
package mapl_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr/localizer/mapl"
)

// TestLocalize_PresentAndAbsent proves the most basic contract of the Localizer:
// a stored (locale, key) returns its template with ok=true, and any missing pair
// returns ("", false) so the herr core can fall back to its own message chain.
func TestLocalize_PresentAndAbsent(t *testing.T) {
	l := mapl.New(map[string]map[string]string{
		"id": {"errors.not_found.message": "Tidak ditemukan."},
	})

	// Present (locale, key) → stored value, ok true.
	if got, ok := l.Localize("id", "errors.not_found.message", nil); !ok || got != "Tidak ditemukan." {
		t.Errorf("Localize(present) = (%q, %v), want (%q, true)", got, ok, "Tidak ditemukan.")
	}

	// Absent key in a known locale → empty, ok false.
	if got, ok := l.Localize("id", "errors.unknown.message", nil); ok || got != "" {
		t.Errorf("Localize(absent key) = (%q, %v), want (\"\", false)", got, ok)
	}

	// Absent locale entirely → empty, ok false.
	if got, ok := l.Localize("en", "errors.not_found.message", nil); ok || got != "" {
		t.Errorf("Localize(absent locale) = (%q, %v), want (\"\", false)", got, ok)
	}
}

// TestLocalize_SubstitutesPlaceholders proves the template fills {name} holes
// from params, and — critically — that substitution is injection-safe: a param
// VALUE that itself contains "{other}" is inserted as literal text and never
// re-scanned, so it cannot trigger a second lookup. This mirrors the core's
// single-pass substituter exactly.
func TestLocalize_SubstitutesPlaceholders(t *testing.T) {
	l := mapl.New(map[string]map[string]string{
		"en": {"greet": "Hello {name}, welcome!"},
		"id": {"inject": "Value: {payload}"},
	})

	// Plain substitution.
	if got, ok := l.Localize("en", "greet", map[string]any{"name": "Ada"}); !ok || got != "Hello Ada, welcome!" {
		t.Errorf("Localize(greet) = (%q, %v), want (%q, true)", got, ok, "Hello Ada, welcome!")
	}

	// Injection safety: the value contains {other} but must NOT be expanded,
	// even though "other" is also a provided param.
	got, ok := l.Localize("id", "inject", map[string]any{
		"payload": "{other}",
		"other":   "EXPANDED",
	})
	if !ok || got != "Value: {other}" {
		t.Errorf("Localize(inject) = (%q, %v), want (%q, true) — value must not be re-expanded", got, ok, "Value: {other}")
	}
}

// TestNew_CopiesInput proves New defensively snapshots its input: mutating the
// caller's maps (or swapping inner entries) after construction must not alter
// what the Localizer returns. This keeps lookups stable and race-free even if a
// caller keeps and edits the table they passed in.
func TestNew_CopiesInput(t *testing.T) {
	inner := map[string]string{"k": "original"}
	tables := map[string]map[string]string{"en": inner}

	l := mapl.New(tables)

	// Mutate the caller's maps after construction.
	inner["k"] = "mutated"             // change an existing inner value
	inner["added"] = "late"            // add a new inner key
	tables["en"] = map[string]string{} // swap the whole inner map
	delete(tables, "en")               // and drop the locale

	if got, ok := l.Localize("en", "k", nil); !ok || got != "original" {
		t.Errorf("Localize after caller mutation = (%q, %v), want (%q, true)", got, ok, "original")
	}
	if got, ok := l.Localize("en", "added", nil); ok || got != "" {
		t.Errorf("Localize(late-added key) = (%q, %v), want (\"\", false)", got, ok)
	}
}
