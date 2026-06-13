package httperr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jeremygeraldprawira/herr"
	"github.com/jeremygeraldprawira/herr/httperr"
)

// decodeBody runs Write against a recorder and returns the recorder plus the decoded JSON
// body, the way a client would receive it.
func decodeBody(t *testing.T, err error, req *http.Request) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	httperr.Write(rec, req, err)
	var body map[string]any
	if e := json.Unmarshal(rec.Body.Bytes(), &body); e != nil {
		t.Fatalf("response body is not valid JSON: %v (raw: %s)", e, rec.Body.String())
	}
	return rec, body
}

// TestWrite_StatusAndBody proves the handler's one-liner contract: Write turns a herr error
// into an HTTP response — the Kind-derived status on the status line and the safe public
// body as JSON.
func TestWrite_StatusAndBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/orders/42", nil)
	e := herr.New("ORDER_NOT_FOUND").Kind(herr.KindNotFound).
		Public(herr.Msg("We couldn't find that order."))

	rec, body := decodeBody(t, e, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if body["code"] != "ORDER_NOT_FOUND" {
		t.Errorf("body.code = %v, want ORDER_NOT_FOUND", body["code"])
	}
	if body["message"] != "We couldn't find that order." {
		t.Errorf("body.message = %v, want the public message", body["message"])
	}
}

// TestWrite_RetryAfterHeader proves a retry delay becomes the protocol-level Retry-After
// header (whole seconds), not just a body field — so standard HTTP clients and proxies can
// honor it. When no delay is set, the header is absent (never "Retry-After: 0").
func TestWrite_RetryAfterHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	e := herr.New("DOWN").Kind(herr.KindUnavailable).RetryAfter(30 * time.Second)
	rec := httptest.NewRecorder()
	httperr.Write(rec, req, e)
	if got := rec.Header().Get("Retry-After"); got != "30" {
		t.Errorf("Retry-After = %q, want %q", got, "30")
	}

	// No delay → header omitted.
	rec2 := httptest.NewRecorder()
	httperr.Write(rec2, req, herr.New("X").Kind(herr.KindNotFound))
	if got := rec2.Header().Get("Retry-After"); got != "" {
		t.Errorf("Retry-After = %q, want it absent", got)
	}
}
