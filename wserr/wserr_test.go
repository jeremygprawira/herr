package wserr_test

import (
	"strings"
	"testing"

	"github.com/jeremygeraldprawira/herr"
	"github.com/jeremygeraldprawira/herr/wserr"
)

// TestClose_CodeAndReason proves the core mapping: a herr error becomes a WebSocket close
// code (from its Kind) and a SAFE, user-facing reason (the public message) — the two pieces
// a server needs to close a connection meaningfully.
func TestClose_CodeAndReason(t *testing.T) {
	e := herr.New("SLOW_DOWN").Kind(herr.KindRateLimited).
		Public(herr.Msg("You're going too fast. Try again shortly."))

	code, reason := wserr.Close(e, "")

	if code != 1013 { // RFC 6455: Try Again Later
		t.Errorf("close code = %d, want 1013", code)
	}
	if reason != "You're going too fast. Try again shortly." {
		t.Errorf("reason = %q, want the public message", reason)
	}
	if strings.Contains(reason, "SLOW_DOWN") {
		// the reason is the human message, not the code
		t.Errorf("reason unexpectedly contains the code: %q", reason)
	}
}
