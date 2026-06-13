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
