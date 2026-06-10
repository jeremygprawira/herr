package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestParam_Substitution proves {name} placeholders in Title/Message are filled from
// params at render time.
func TestParam_Substitution(t *testing.T) {
	e := herr.New("INSUFFICIENT").
		Public(herr.Public{
			Title:   "Balance too low",
			Message: "Your balance {available} is below the required {min}.",
		}).
		Param("available", "Rp3.2M").
		Param("min", "Rp5M")

	body := decodeWire(t, e)
	if body["message"] != "Your balance Rp3.2M is below the required Rp5M." {
		t.Errorf("message = %q", body["message"])
	}
}

// TestParam_NoFormatInjection is the H3 security property: a param VALUE that itself
// contains format verbs (%s) or brace tokens ({evil}) must be inserted verbatim and
// never interpreted — no Sprintf injection, no re-scanning of substituted text.
func TestParam_NoFormatInjection(t *testing.T) {
	e := herr.New("X").
		Public(herr.Msg("Hello {name}!")).
		Param("name", "%s%d {evil} 100%").
		Param("evil", "PWNED") // must NOT be used to fill the {evil} that came from a value

	body := decodeWire(t, e)
	want := "Hello %s%d {evil} 100%!"
	if body["message"] != want {
		t.Errorf("message = %q, want %q (injection!)", body["message"], want)
	}
}

// TestParam_MissingRendersEmpty proves an unfilled placeholder collapses to empty in the
// default (production) mode rather than showing raw `{name}` braces to a user. StrictMode
// (added later) will turn this into a test-time failure so it never ships.
func TestParam_MissingRendersEmpty(t *testing.T) {
	e := herr.New("X").Public(herr.Msg("A{missing}B"))

	body := decodeWire(t, e)
	if body["message"] != "AB" {
		t.Errorf("message = %q, want %q", body["message"], "AB")
	}
}
