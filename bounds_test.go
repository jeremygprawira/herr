package herr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestBounds_FieldsCapped proves internal fields can't grow without limit: past the cap,
// further additions are dropped and a single truncation marker is recorded. This bounds
// memory and log-line size even if a buggy loop or an attacker drives many .With calls.
func TestBounds_FieldsCapped(t *testing.T) {
	e := herr.New("X")
	for i := 0; i < 500; i++ {
		e.With(fmt.Sprintf("k%d", i), i)
	}

	rec := herr.LogRecord(e)
	if len(rec.Fields) > 70 { // cap is 64 + at most one marker
		t.Errorf("internal fields not capped: got %d", len(rec.Fields))
	}
	var marker bool
	for _, f := range rec.Fields {
		if f.Key == "_fields_truncated" {
			marker = true
		}
	}
	if !marker {
		t.Error("expected a _fields_truncated marker once the cap is hit")
	}
}

// TestBounds_MetadataCapped proves the same protection for public metadata.
func TestBounds_MetadataCapped(t *testing.T) {
	e := herr.New("X")
	for i := 0; i < 500; i++ {
		e.WithPublic(fmt.Sprintf("k%d", i), i)
	}

	meta := decodeWire(t, e)["metadata"].(map[string]any)
	if len(meta) > 70 {
		t.Errorf("public metadata not capped: got %d", len(meta))
	}
}

// TestBounds_InternalMessageTruncated proves a single huge internal message can't blow up
// a log line: past the per-string cap it is truncated and a visible marker is appended, so
// the loss is obvious rather than silent.
func TestBounds_InternalMessageTruncated(t *testing.T) {
	long := strings.Repeat("x", 100_000)
	e := herr.New("BOOM").Internal(long)

	got := herr.LogRecord(e).Internal
	if len(got) >= len(long) {
		t.Fatalf("internal message not truncated: got len %d", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("truncated internal message missing its marker: ...%q", got[len(got)-40:])
	}
}

// TestBounds_FieldStringTruncated proves a huge string VALUE in an internal field is
// truncated the same way the internal message is — so one .With call with a giant blob
// can't blow up a log line. Non-string values are left untouched.
func TestBounds_FieldStringTruncated(t *testing.T) {
	long := strings.Repeat("y", 100_000)
	e := herr.New("X").With("blob", long)

	var got string
	for _, f := range herr.LogRecord(e).Fields {
		if f.Key == "blob" {
			got, _ = f.Val.(string)
		}
	}
	if got == "" {
		t.Fatal("blob field missing")
	}
	if len(got) >= len(long) {
		t.Fatalf("field value not truncated: got len %d", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("truncated field value missing its marker: ...%q", got[len(got)-40:])
	}
}

// TestBounds_MetadataStringTruncated proves a huge PUBLIC metadata string is truncated
// before it crosses the wire — bounding response size, not just log size. Public matters
// most here: an unbounded metadata value would inflate every client response.
func TestBounds_MetadataStringTruncated(t *testing.T) {
	long := strings.Repeat("z", 100_000)
	e := herr.New("X").WithPublic("note", long)

	meta := decodeWire(t, e)["metadata"].(map[string]any)
	got, _ := meta["note"].(string)
	if got == "" {
		t.Fatal("note metadata missing")
	}
	if len(got) >= len(long) {
		t.Fatalf("metadata value not truncated: got len %d", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("truncated metadata value missing its marker: ...%q", got[len(got)-40:])
	}
}
