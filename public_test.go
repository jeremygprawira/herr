package herr_test

import (
	"encoding/json"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// decodeWire marshals an *Error the way a transport would (MarshalJSON delegates to the
// safe wire DTO) and decodes it into a generic map so tests can assert on exactly which
// keys/values reach the client. This is the lens through which we verify BOTH that
// public data is present (here) and that internal data is absent (the leak gate).
func decodeWire(t *testing.T, e *herr.Error) map[string]any {
	t.Helper()
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal(*Error) failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decoding wire body failed: %v (body=%s)", err, raw)
	}
	return m
}

// TestPublic_RendersToWireBody proves the public surface — Title, Message, and the
// free-form Metadata bag — crosses the wire under stable JSON keys.
func TestPublic_RendersToWireBody(t *testing.T) {
	e := herr.New("ACCOUNT_CONNECT_FAILED").
		Kind(herr.KindUnavailable).
		Public(herr.Public{
			Title:    "Unable to connect your account",
			Message:  "We couldn't connect your account due to a technical issue on our end.",
			Metadata: map[string]any{"support_url": "https://example.com/support"},
		})

	body := decodeWire(t, e)

	if body["code"] != "ACCOUNT_CONNECT_FAILED" {
		t.Errorf("code = %v, want ACCOUNT_CONNECT_FAILED", body["code"])
	}
	if body["title"] != "Unable to connect your account" {
		t.Errorf("title = %v", body["title"])
	}
	if body["message"] != "We couldn't connect your account due to a technical issue on our end." {
		t.Errorf("message = %v", body["message"])
	}
	meta, ok := body["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or wrong type: %v", body["metadata"])
	}
	if meta["support_url"] != "https://example.com/support" {
		t.Errorf("metadata.support_url = %v", meta["support_url"])
	}
}

// TestMsg_Shorthand proves herr.Msg(s) is sugar for a Public carrying just a Message, so
// the common "I only have one sentence" case stays a one-liner.
func TestMsg_Shorthand(t *testing.T) {
	e := herr.New("NOT_FOUND").Public(herr.Msg("Resource not found."))

	body := decodeWire(t, e)
	if body["message"] != "Resource not found." {
		t.Errorf("message = %v, want %q", body["message"], "Resource not found.")
	}
	// Title/metadata were never set, so they must be omitted (not empty strings/objects).
	if _, present := body["title"]; present {
		t.Errorf("title should be omitted when unset, got %v", body["title"])
	}
}
