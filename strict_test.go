package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestStrictMode_MissingParamVisible proves the H3 dev-aid: by default a missing {name}
// placeholder collapses to empty (a polished, if slightly odd, production message), but in
// StrictMode the unfilled placeholder is left VISIBLE so a developer immediately sees they
// forgot to supply a Param — never a panic, just a loud, obvious gap.
func TestStrictMode_MissingParamVisible(t *testing.T) {
	e := func() *herr.Error {
		return herr.New("GREETING").Public(herr.Msg("Hello {name}, welcome back!"))
	}

	// Default (production): the missing param disappears.
	if got := decodeWire(t, e())["message"]; got != "Hello , welcome back!" {
		t.Errorf("default message = %q, want the placeholder collapsed to empty", got)
	}

	// StrictMode: the missing param is visible.
	herr.StrictMode(true)
	t.Cleanup(func() { herr.StrictMode(false) })
	if got := decodeWire(t, e())["message"]; got != "Hello {name}, welcome back!" {
		t.Errorf("strict message = %q, want the unfilled placeholder left visible", got)
	}

	// A SUPPLIED param is substituted normally in either mode.
	if got := decodeWire(t, e().Param("name", "Sam"))["message"]; got != "Hello Sam, welcome back!" {
		t.Errorf("strict message with param = %q, want it substituted", got)
	}
}
