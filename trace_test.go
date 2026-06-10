package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestTrace proves a correlation id, when set, is exposed under `traceId` so a user can
// quote it to support and an operator can find the exact request in the logs — the API
// equivalent of "offer a path to support".
func TestTrace(t *testing.T) {
	e := herr.New("BOOM").Trace("f1c2-abc123")
	if got := decodeWire(t, e)["traceId"]; got != "f1c2-abc123" {
		t.Errorf("traceId = %v, want f1c2-abc123", got)
	}
}

// TestTrace_OmittedWhenUnset proves we never emit an empty traceId; the transport layer
// is responsible for injecting one when appropriate.
func TestTrace_OmittedWhenUnset(t *testing.T) {
	if _, present := decodeWire(t, herr.New("BOOM"))["traceId"]; present {
		t.Error("traceId should be omitted when unset")
	}
}
