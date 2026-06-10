package herr_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestCatalog_NewCarriesDefaults proves the "catalog" authoring style: define an error
// class once, then .New() produces a fresh *Error pre-filled with that class's defaults
// (code, Kind-derived status, public message).
func TestCatalog_NewCarriesDefaults(t *testing.T) {
	var ErrNotFound = herr.Define(herr.Class{
		Code:   "NOT_FOUND",
		Kind:   herr.KindNotFound,
		Public: herr.Public{Message: "Resource not found."},
	})

	e := ErrNotFound.New()

	if e.Code() != "NOT_FOUND" {
		t.Errorf("Code() = %q", e.Code())
	}
	if e.HTTPStatus() != 404 {
		t.Errorf("HTTPStatus() = %d, want 404", e.HTTPStatus())
	}
	if body := decodeWire(t, e); body["message"] != "Resource not found." {
		t.Errorf("message = %v", body["message"])
	}
}

// TestCatalog_InstancesAreIsolated is the heart of C1: two errors built from the SAME
// catalog entry must not see each other's per-request data, and the shared class
// metadata must still appear in both. This proves .New() does not share mutable state.
func TestCatalog_InstancesAreIsolated(t *testing.T) {
	var ErrX = herr.Define(herr.Class{
		Code:   "X",
		Public: herr.Public{Metadata: map[string]any{"base": "shared"}},
	})

	a := ErrX.New().WithPublic("req", "A")
	b := ErrX.New().WithPublic("req", "B")

	metaA := decodeWire(t, a)["metadata"].(map[string]any)
	metaB := decodeWire(t, b)["metadata"].(map[string]any)

	if metaA["req"] != "A" {
		t.Errorf("A.req = %v, want A", metaA["req"])
	}
	if metaB["req"] != "B" {
		t.Errorf("B.req = %v, want B (cross-contamination!)", metaB["req"])
	}
	// The class-level metadata must reach both instances...
	if metaA["base"] != "shared" || metaB["base"] != "shared" {
		t.Errorf("shared class metadata missing: A=%v B=%v", metaA["base"], metaB["base"])
	}
	// ...and must NOT have been polluted by either instance's per-request additions.
	if _, leaked := metaB["req"]; metaB["req"] == "A" {
		t.Errorf("A's data leaked into B: %v", leaked)
	}
}

// TestCatalog_ConcurrentNewIsRaceFree runs the race detector against many goroutines all
// building + rendering errors from a single catalog entry simultaneously. Run with
// `go test -race`; it must report no data races and no panics.
func TestCatalog_ConcurrentNewIsRaceFree(t *testing.T) {
	var ErrX = herr.Define(herr.Class{
		Code:   "X",
		Kind:   herr.KindInternal,
		Public: herr.Public{Message: "boom", Metadata: map[string]any{"base": "shared"}},
	})

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e := ErrX.New().WithPublic("i", i).With("secret", i)
			if _, err := json.Marshal(e); err != nil {
				t.Errorf("marshal failed: %v", err)
			}
		}(i)
	}
	wg.Wait()
}
