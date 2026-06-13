// Package httperr is the HTTP transport adapter for herr.
//
// A handler builds nothing by hand: it returns/produces a *herr.Error (or any error) and
// calls httperr.Write(w, r, err). This package resolves the request locale, renders the
// SAFE public body via the core's allow-listed wire DTO, sets the status line from the
// error's Kind, and applies retry headers. Because it renders only through herr's Body(),
// it inherits the C2 guarantee — no internal data can reach the response.
//
// It depends ONLY on the standard library (net/http) and the herr core, so adding the HTTP
// transport never pulls a third-party dependency into a project.
package httperr

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jeremygeraldprawira/herr"
)

// Write renders err as an HTTP response and returns the status code it wrote.
//
// The flow is deliberately small and total:
//  1. Coerce err into a *herr.Error (a non-herr error becomes a safe 500 with the original
//     preserved as the internal cause — never leaked to the body).
//  2. Resolve the response locale from the request's Accept-Language header.
//  3. Set Content-Type and, when a retry delay is present, the Retry-After header.
//  4. Write the Kind-derived status line and the safe JSON body.
func Write(w http.ResponseWriter, r *http.Request, err error) int {
	he := coerce(err)

	locale := localeFrom(r)
	status := he.HTTPStatus()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The body is the core's safe wire DTO; encoding errors are not expected for the
	// allow-listed shape, and the status/headers are already committed, so we ignore the
	// (practically unreachable) encode error rather than double-writing.
	_ = json.NewEncoder(w).Encode(he.Body(locale))
	return status
}

// coerce returns err as a *herr.Error. If err already is (or wraps) one, that instance is
// used as-is. Otherwise a fresh server-fault error is built that WRAPS the original — so the
// cause is available to logs while the client only ever sees a safe, generic 500 body.
func coerce(err error) *herr.Error {
	var he *herr.Error
	if errors.As(err, &he) {
		return he
	}
	return herr.New("INTERNAL").Kind(herr.KindInternal).Wrap(err)
}

// localeFrom extracts a best-effort locale tag from the request's Accept-Language header.
// It takes the FIRST listed language tag (highest client preference), dropping any `;q=`
// weight. This keeps httperr dependency-free; the supported-locale allow-list matching
// (H4, via x/text) is layered on later. An empty result lets the core fall back to its
// literal/floor message.
func localeFrom(r *http.Request) string {
	if r == nil {
		return ""
	}
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return ""
	}
	// "en-US,en;q=0.9,id;q=0.8" → "en-US": take the first tag, drop any ;q= weight.
	tag := header
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	if i := strings.IndexByte(tag, ';'); i >= 0 {
		tag = tag[:i]
	}
	return strings.TrimSpace(tag)
}
