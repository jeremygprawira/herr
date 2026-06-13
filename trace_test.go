package herr_test

import (
	"strings"
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

// TestNewTraceID proves the helper transports use to mint a correlation id when the caller
// supplied none: it returns a non-empty, fixed-length hex id, and successive calls are
// unique (so two concurrent requests never collide).
func TestNewTraceID(t *testing.T) {
	const hexChars = "0123456789abcdef"

	id := herr.NewTraceID()
	if len(id) != 32 { // 16 random bytes, hex-encoded
		t.Errorf("len(NewTraceID()) = %d, want 32", len(id))
	}
	for _, r := range id {
		if !strings.ContainsRune(hexChars, r) {
			t.Fatalf("NewTraceID() = %q contains non-hex char %q", id, r)
		}
	}

	// Uniqueness across a batch — randomness, not a constant.
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := herr.NewTraceID()
		if seen[id] {
			t.Fatalf("NewTraceID() produced a duplicate: %q", id)
		}
		seen[id] = true
	}
}
