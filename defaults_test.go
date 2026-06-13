package herr_test

import (
	"strings"
	"testing"

	"github.com/jeremygprawira/herr"
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

// TestSetDefaults_OverridesFloor proves the floor itself is overridable: SetDefaults
// installs a process-wide map that replaces the built-in floor message for the Kinds it
// names. Kinds it does NOT name keep their built-in floor, and SetDefaults(nil) restores
// the built-in set entirely. The override is the FLOOR only — an explicit message still wins.
func TestSetDefaults_OverridesFloor(t *testing.T) {
	herr.SetDefaults(map[herr.Kind]string{
		herr.KindNotFound: "Nothing here. (custom)",
	})
	t.Cleanup(func() { herr.SetDefaults(nil) })

	// The named Kind uses the override.
	if got := decodeWire(t, herr.New("X").Kind(herr.KindNotFound))["message"]; got != "Nothing here. (custom)" {
		t.Errorf("NotFound floor = %q, want the custom override", got)
	}

	// An unnamed Kind keeps its built-in floor.
	internal, _ := decodeWire(t, herr.New("BOOM"))["message"].(string)
	if !strings.Contains(strings.ToLower(internal), "went wrong") {
		t.Errorf("Internal floor = %q, want the built-in (unoverridden) message", internal)
	}

	// An explicit message still beats the override — the override is only the floor.
	e := herr.New("X").Kind(herr.KindNotFound).Public(herr.Msg("Explicit."))
	if got := decodeWire(t, e)["message"]; got != "Explicit." {
		t.Errorf("message = %q, want the explicit one to win over the override", got)
	}
}
