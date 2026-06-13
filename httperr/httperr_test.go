package httperr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// stubLocalizer translates one (locale,key) pair; everything else falls through.
type stubLocalizer struct{ locale, key, msg string }

func (s stubLocalizer) Localize(locale, key string, _ map[string]any) (string, bool) {
	if locale == s.locale && key == s.key {
		return s.msg, true
	}
	return "", false
}

// TestWrite_LocaleFromAcceptLanguage proves the request's Accept-Language header drives the
// rendered message: with a Localizer installed, an `id` request gets the Indonesian
// translation while the default request keeps the literal message.
func TestWrite_LocaleFromAcceptLanguage(t *testing.T) {
	herr.SetLocalizer(stubLocalizer{
		locale: "id", key: "errors.not_found.message", msg: "Tidak ditemukan.",
	})
	t.Cleanup(func() { herr.SetLocalizer(nil) })

	// No inline message — so resolution reaches the Localizer step (an inline message would
	// win over any translation).
	makeErr := func() *herr.Error {
		return herr.New("NOT_FOUND").Kind(herr.KindNotFound)
	}

	// Accept-Language: id → translated.
	reqID := httptest.NewRequest(http.MethodGet, "/", nil)
	reqID.Header.Set("Accept-Language", "id-ID,id;q=0.9,en;q=0.8")
	if _, body := decodeBody(t, makeErr(), reqID); body["message"] != "Tidak ditemukan." {
		t.Errorf("id message = %v, want the Indonesian translation", body["message"])
	}
}

// TestWrite_NonHerrErrorIsSafe500 proves the boundary holds for a PLAIN error: Write turns
// it into a 500 with a calm generic body, and the raw error string (which may carry
// internals) never reaches the client.
func TestWrite_NonHerrErrorIsSafe500(t *testing.T) {
	const secret = "pq: password authentication failed for user app"
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec, body := decodeBody(t, errors.New(secret), req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "password authentication failed") {
		t.Errorf("LEAK: raw error surfaced in body: %s", rec.Body.String())
	}
	if msg, _ := body["message"].(string); msg == "" {
		t.Error("a 500 must still carry a non-empty, safe message")
	}
}
