package herr_test

import (
	"fmt"
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
