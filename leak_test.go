package herr_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jeremygprawira/herr"
)

// TestSafeSplit_InternalNeverLeaks is the C2 security gate.
//
// It stuffs a unique secret marker into EVERY internal channel — the internal message,
// an internal field, and a wrapped cause — then serializes the error the way a client
// would receive it and asserts none of those markers appear. This is the single most
// important test in the library: it proves the public/internal boundary holds.
func TestSafeSplit_InternalNeverLeaks(t *testing.T) {
	const (
		secretMsg   = "SECRET_INTERNAL_MESSAGE_pq_deadlock"
		secretField = "SECRET_FIELD_VALUE_jdbc_password"
		secretCause = "SECRET_CAUSE_connection_string"
	)

	e := herr.New("ACCOUNT_CONNECT_FAILED").
		Kind(herr.KindUnavailable).
		Public(herr.Msg("We couldn't connect your account.")).
		Internal(secretMsg).               // developer-only message
		With("db_dsn", secretField).       // internal structured field
		Wrap(errors.New(secretCause)).     // underlying cause
		WithStack()                        // captured stack (server-fault kind)

	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body := string(raw)

	for _, secret := range []string{secretMsg, secretField, secretCause} {
		if strings.Contains(body, secret) {
			t.Errorf("LEAK: internal secret %q appeared in wire body: %s", secret, body)
		}
	}

	// The captured stack is INTERNAL too: no frame (file:line, package path) may surface.
	// `.go:` is a reliable, content-independent marker of a leaked stack frame.
	if strings.Contains(body, ".go:") {
		t.Errorf("LEAK: a captured stack frame appeared in wire body: %s", body)
	}

	// The public message, by contrast, MUST be present.
	if !strings.Contains(body, "We couldn't connect your account.") {
		t.Errorf("public message missing from wire body: %s", body)
	}
}

// TestWithPublic_AppearsInMetadata proves the OTHER side of the split: data the developer
// explicitly marks public via WithPublic lands in the wire metadata bag.
func TestWithPublic_AppearsInMetadata(t *testing.T) {
	e := herr.New("RATE_LIMITED").
		Kind(herr.KindRateLimited).
		WithPublic("retry_after_seconds", 30)

	body := decodeWire(t, e)
	meta, ok := body["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing: %v", body)
	}
	// JSON numbers decode to float64.
	if meta["retry_after_seconds"] != float64(30) {
		t.Errorf("metadata.retry_after_seconds = %v, want 30", meta["retry_after_seconds"])
	}
}

// TestInternal_AppearsInErrorString proves internal detail still serves DEVELOPERS: the
// Error() string (used in logs) includes the internal message even though it never
// reaches the client.
func TestInternal_AppearsInErrorString(t *testing.T) {
	e := herr.New("BOOM").Internal("low-level: connection refused")
	if !strings.Contains(e.Error(), "connection refused") {
		t.Errorf("Error() = %q, want it to include the internal detail", e.Error())
	}
}

// FuzzWire_NeverLeaksInternal hammers the safe split with arbitrary inputs: whatever
// random bytes go into the internal channels, they must never surface in the serialized
// body. This catches leaks that hand-picked examples might miss.
func FuzzWire_NeverLeaksInternal(f *testing.F) {
	f.Add("code", "internal detail", "field value", "cause text")
	f.Fuzz(func(t *testing.T, code, internal, field, cause string) {
		if internal == "" && field == "" && cause == "" {
			return // nothing secret to leak
		}
		e := herr.New(code).
			Public(herr.Msg("safe public message")).
			Internal(internal).
			With("k", field).
			Wrap(errors.New(cause))

		raw, err := json.Marshal(e)
		if err != nil {
			t.Skip() // marshal error is not a leak; out of scope for this property
		}
		body := string(raw)

		// The body legitimately contains PUBLIC content — the code and the public
		// message — in their JSON-encoded (escaped) form. A secret that appears only as a
		// substring of that encoded public content is NOT a leak; it is coincidental
		// overlap (e.g. a NUL byte in `code` JSON-escapes to the literal six-character
		// sequence backslash-u-0-0-0-0, whose text contains "0000"). Build the encoded
		// public strings so we can discount such matches and
		// keep the gate honest: any secret appearing OUTSIDE the public content still trips.
		encCode, _ := json.Marshal(code)
		encMsg, _ := json.Marshal("safe public message")
		publicEncoded := string(encCode) + string(encMsg)

		for _, secret := range []string{internal, field, cause} {
			// Only meaningful, non-trivial secrets can "leak"; skip values that are too
			// short to be distinctive or are fully explained by the encoded public content.
			if len(secret) < 4 || strings.Contains(publicEncoded, secret) {
				continue
			}
			if strings.Contains(body, secret) {
				t.Errorf("LEAK: %q surfaced in body %s", secret, body)
			}
		}
	})
}
