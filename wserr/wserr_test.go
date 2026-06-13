package wserr_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

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

// TestControlPayload_Encoding proves the ready-to-send form: the close-frame payload is the
// 2-byte big-endian close code followed by the UTF-8 reason (RFC 6455 §5.5.1), so a caller
// can hand it straight to their WebSocket library's close/control-write method.
func TestControlPayload_Encoding(t *testing.T) {
	e := herr.New("DOWN").Kind(herr.KindUnavailable).Public(herr.Msg("Down for maintenance."))

	payload := wserr.ControlPayload(e, "")

	if len(payload) < 2 {
		t.Fatalf("payload too short: %d bytes", len(payload))
	}
	code := int(payload[0])<<8 | int(payload[1])
	if code != 1013 { // KindUnavailable → Try Again Later
		t.Errorf("encoded close code = %d, want 1013", code)
	}
	if string(payload[2:]) != "Down for maintenance." {
		t.Errorf("encoded reason = %q, want the public message", string(payload[2:]))
	}
	if len(payload) > 125 { // control-frame payload hard limit
		t.Errorf("payload = %d bytes, want <= 125", len(payload))
	}
}

// TestClose_ReasonTruncatedToFrameLimit proves a long public message can't produce an
// invalid (oversized) close frame: the reason is bounded to 123 bytes on a UTF-8 boundary.
func TestClose_ReasonTruncatedToFrameLimit(t *testing.T) {
	long := strings.Repeat("é", 200) // 400 bytes of valid UTF-8
	e := herr.New("X").Kind(herr.KindInvalid).Public(herr.Msg(long))

	_, reason := wserr.Close(e, "")
	if len(reason) > 123 {
		t.Errorf("reason = %d bytes, want <= 123", len(reason))
	}
	if !utf8.ValidString(reason) {
		t.Error("truncated reason is not valid UTF-8 (cut mid-rune)")
	}
}

// TestClose_NonHerrIsSafe proves a plain error yields the internal-error close code (1011)
// and never leaks its raw string into the reason.
func TestClose_NonHerrIsSafe(t *testing.T) {
	code, reason := wserr.Close(errors.New("raw secret: db dsn leaked"), "")
	if code != 1011 { // RFC 6455 Internal Error
		t.Errorf("close code = %d, want 1011", code)
	}
	if strings.Contains(reason, "db dsn leaked") {
		t.Errorf("LEAK: raw error surfaced in reason: %q", reason)
	}
}
