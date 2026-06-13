package herr_test

import (
	"encoding/json"
	"testing"

	"github.com/jeremygprawira/herr"
)

// mapLocalizer is a trivial test Localizer keyed by "locale|key". It stands in for any
// real i18n library — herr only depends on the Localizer interface, never a concrete one.
type mapLocalizer map[string]string

func (m mapLocalizer) Localize(locale, key string, _ map[string]any) (string, bool) {
	s, ok := m[locale+"|"+key]
	return s, ok
}

// bodyMessage renders the error for a specific locale (the way a transport would) and
// returns the resolved public message.
func bodyMessage(t *testing.T, e *herr.Error, locale string) string {
	t.Helper()
	raw, err := json.Marshal(e.Body(locale))
	if err != nil {
		t.Fatalf("marshal Body failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	s, _ := m["message"].(string)
	return s
}

// TestLocalizer_TranslatesByDerivedKey proves the i18n magic: with no explicit key, herr
// derives `errors.<code>.message` and asks the Localizer. A translation is used when
// present; otherwise it falls back to the literal catalog message (never a raw key).
func TestLocalizer_TranslatesByDerivedKey(t *testing.T) {
	herr.SetLocalizer(mapLocalizer{
		"id|errors.not_found.message": "Tidak ditemukan.",
	})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	var ErrNotFound = herr.Define(herr.Class{
		Code:   "NOT_FOUND",
		Kind:   herr.KindNotFound,
		Public: herr.Public{Message: "Resource not found."},
	})

	if got := bodyMessage(t, ErrNotFound.New(), "id"); got != "Tidak ditemukan." {
		t.Errorf("id message = %q, want the Indonesian translation", got)
	}
	if got := bodyMessage(t, ErrNotFound.New(), "en"); got != "Resource not found." {
		t.Errorf("en message = %q, want the literal fallback", got)
	}
}

// TestMessageKey_OverridesDerivedKey proves the explicit message-key override: by default
// herr asks the Localizer for the DERIVED key (`errors.<code>.message`), but a developer
// can point an error at a different i18n key with .MessageKey(k). The Localizer is then
// asked for that exact key — letting several codes share one translation, or a code map to
// a key that doesn't follow the naming convention.
func TestMessageKey_OverridesDerivedKey(t *testing.T) {
	herr.SetLocalizer(mapLocalizer{
		// Only the CUSTOM key has a translation; the derived key does not.
		"id|errors.shared.not_available": "Tidak tersedia.",
	})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	// No inline message — so the resolver reaches the Localizer step and uses the key.
	e := herr.New("WIDGET_GONE").MessageKey("errors.shared.not_available")

	if got := bodyMessage(t, e, "id"); got != "Tidak tersedia." {
		t.Errorf("id message = %q, want the translation under the explicit key", got)
	}
}

// TestClassMessageKey_OverridesDerivedKey proves the explicit key works from the CATALOG
// too: a Class can declare MessageKey once, and every error stamped via .New() asks the
// Localizer for that key instead of the derived one. The catalog message stays the literal
// fallback when no translation exists.
func TestClassMessageKey_OverridesDerivedKey(t *testing.T) {
	herr.SetLocalizer(mapLocalizer{
		"id|errors.shared.not_available": "Tidak tersedia.",
	})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	var ErrWidgetGone = herr.Define(herr.Class{
		Code:       "WIDGET_GONE",
		Kind:       herr.KindNotFound,
		MessageKey: "errors.shared.not_available",
		Public:     herr.Public{Message: "Widget not available."},
	})

	if got := bodyMessage(t, ErrWidgetGone.New(), "id"); got != "Tidak tersedia." {
		t.Errorf("id message = %q, want the translation under the class's explicit key", got)
	}
	if got := bodyMessage(t, ErrWidgetGone.New(), "en"); got != "Widget not available." {
		t.Errorf("en message = %q, want the literal catalog fallback", got)
	}
}

// TestLocalizer_InlineMessageWins proves an inline call-site message overrides any
// translation — when you write the exact words at the call site, you mean them.
func TestLocalizer_InlineMessageWins(t *testing.T) {
	herr.SetLocalizer(mapLocalizer{"id|errors.x.message": "translated"})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	e := herr.New("X").Public(herr.Msg("inline override"))
	if got := bodyMessage(t, e, "id"); got != "inline override" {
		t.Errorf("message = %q, want inline override to win", got)
	}
}
