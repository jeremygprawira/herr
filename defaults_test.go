package herr_test

import (
	"strings"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestDefaultMessage_FloorByKind proves the safety floor: an error with no explicit
// public message still renders a calm, Kind-appropriate sentence (never blank, never a
// leak). This is the library's "you can't ship an ugly error by forgetting" guarantee.
func TestDefaultMessage_FloorByKind(t *testing.T) {
	notFound := decodeWire(t, herr.New("X").Kind(herr.KindNotFound))["message"]
	if s, _ := notFound.(string); !strings.Contains(strings.ToLower(s), "couldn't find") {
		t.Errorf("NotFound default = %q, want it to mention not finding", notFound)
	}

	// KindInternal (the zero value) → the own-it server-fault message.
	internal := decodeWire(t, herr.New("BOOM"))["message"]
	if s, _ := internal.(string); !strings.Contains(strings.ToLower(s), "went wrong") {
		t.Errorf("Internal default = %q, want it to own the failure", internal)
	}

	// No error ever renders an empty message.
	if s, _ := internal.(string); s == "" {
		t.Error("default message must never be empty")
	}
}

// TestDefaultMessage_ExplicitWins proves an explicit message always overrides the floor.
func TestDefaultMessage_ExplicitWins(t *testing.T) {
	e := herr.New("X").Kind(herr.KindNotFound).Public(herr.Msg("That order doesn't exist."))
	if got := decodeWire(t, e)["message"]; got != "That order doesn't exist." {
		t.Errorf("message = %q, want the explicit one", got)
	}
}
