package herr_test

import (
	"encoding/json"
	"testing"

	"github.com/jeremygeraldprawira/herr"
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
