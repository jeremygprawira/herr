package herr_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jeremygprawira/herr"
)

// fieldErrorEntries renders the error for a locale and returns its `errors[]` as decoded
// maps, the way a transport + client would see them.
func fieldErrorEntries(t *testing.T, e *herr.Error, locale string) []map[string]any {
	t.Helper()
	raw, err := json.Marshal(e.Body(locale))
	if err != nil {
		t.Fatalf("marshal Body failed: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	arr, _ := body["errors"].([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		m, _ := item.(map[string]any)
		out = append(out, m)
	}
	return out
}

// TestFieldError_RendersErrorsArray proves a parent error can carry per-field children that
// render as a typed, top-level `errors[]` in the wire body (NOT buried in metadata). Each
// entry exposes only the safe public triple {field, code, message} — the shape a front end
// renders next to the offending input.
func TestFieldError_RendersErrorsArray(t *testing.T) {
	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).
		FieldError("email", "INVALID_EMAIL", "Enter a valid email address.").
		FieldError("age", "OUT_OF_RANGE", "Age must be 18 or older.")

	body := decodeWire(t, e)

	arr, ok := body["errors"].([]any)
	if !ok {
		t.Fatalf("errors array missing or wrong type: %T (%v)", body["errors"], body["errors"])
	}
	if len(arr) != 2 {
		t.Fatalf("errors len = %d, want 2", len(arr))
	}

	first, _ := arr[0].(map[string]any)
	if first["field"] != "email" || first["code"] != "INVALID_EMAIL" ||
		first["message"] != "Enter a valid email address." {
		t.Errorf("first entry = %v, want {email, INVALID_EMAIL, Enter a valid email address.}", first)
	}
	second, _ := arr[1].(map[string]any)
	if second["field"] != "age" || second["code"] != "OUT_OF_RANGE" {
		t.Errorf("second entry = %v, want {age, OUT_OF_RANGE, ...}", second)
	}

	// The field errors live at the top level, never inside the public metadata bag.
	if meta, present := body["metadata"].(map[string]any); present {
		if _, leaked := meta["errors"]; leaked {
			t.Error("field errors must be top-level, not inside metadata")
		}
	}
}

// TestFieldError_MessageLocalizes proves each field error's message flows through the SAME
// resolution chain as the top-level message: a Localizer translation keyed by the child's
// code wins, and the literal message is the fallback when none exists.
func TestFieldError_MessageLocalizes(t *testing.T) {
	herr.SetLocalizer(mapLocalizer{
		"id|errors.invalid_email.message": "Masukkan email yang valid.",
	})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).
		FieldError("email", "INVALID_EMAIL", "Enter a valid email address.")

	if got := fieldErrorEntries(t, e, "id")[0]["message"]; got != "Masukkan email yang valid." {
		t.Errorf("id field message = %q, want the Indonesian translation", got)
	}
	if got := fieldErrorEntries(t, e, "en")[0]["message"]; got != "Enter a valid email address." {
		t.Errorf("en field message = %q, want the literal fallback", got)
	}
}

// TestFieldError_InternalDetailNeverLeaks is the C2 guard for the errors[] channel:
// per-field internal detail (a rejected value, a validator reason) added via .With must
// stay in the logs and NEVER surface in the public errors[] — even though the field error
// and the internal detail describe the same field.
func TestFieldError_InternalDetailNeverLeaks(t *testing.T) {
	const secret = "SECRET_rejected_value_user@example.com"

	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).
		With("email_rejected_value", secret). // internal, logs-only
		FieldError("email", "INVALID_EMAIL", "Enter a valid email address.")

	body := decodeWire(t, e)
	raw, _ := json.Marshal(body)
	if strings.Contains(string(raw), secret) {
		t.Errorf("LEAK: internal rejected value surfaced in wire body: %s", string(raw))
	}

	// ...but it IS available to logs.
	if herr.LogFields(e)["email_rejected_value"] != secret {
		t.Error("internal rejected value should be present in the log fields")
	}
}

// TestFieldError_CappedAtLimit proves the H5 bound: no matter how many field errors are
// appended, the rendered errors[] stays bounded and records a visible truncation marker.
func TestFieldError_CappedAtLimit(t *testing.T) {
	e := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable)
	for i := 0; i < 250; i++ {
		e.FieldError("f", "BAD", "nope")
	}

	entries := fieldErrorEntries(t, e, "en")
	if len(entries) > 105 { // cap is 100 + at most one marker
		t.Errorf("errors[] not capped: got %d entries", len(entries))
	}
	var marker bool
	for _, en := range entries {
		if en["code"] == "_errors_truncated" {
			marker = true
		}
	}
	if !marker {
		t.Error("expected an _errors_truncated marker once the cap is hit")
	}
}
