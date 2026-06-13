// Package wserr is the WebSocket transport adapter for herr.
//
// A WebSocket server can't return an HTTP status mid-stream; when it must end a connection
// because of an error it sends a CLOSE control frame carrying a numeric close code and a
// short UTF-8 reason. This package turns a herr error into exactly that pair — the
// Kind-derived close code and a SAFE, user-facing reason — without importing any WebSocket
// library, so it composes with whatever ws stack a project already uses.
//
// Like the rest of herr, it renders only through the core's safe public surface, so no
// internal detail can ride along into the reason sent to the client (C2).
package wserr

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/jeremygprawira/herr"
)

// maxReasonBytes is the RFC 6455 limit on a close-frame reason. A control frame payload is
// at most 125 bytes; the close code takes 2, leaving 123 for the UTF-8 reason text.
const maxReasonBytes = 123

// Close maps err to the WebSocket close code and reason a server should send.
//
// Flow: coerce any error to a *herr.Error (a non-herr error becomes a safe internal-error
// close), take the Kind-derived close code, and use the localized PUBLIC message as the
// reason — truncated to the 123-byte frame limit on a UTF-8 boundary so the control frame
// is always valid.
func Close(err error, locale string) (code int, reason string) {
	he := coerce(err)
	return he.WSClose(), capReason(publicMessage(he, locale))
}

// ControlPayload returns the ready-to-send CLOSE control-frame payload for err: the 2-byte
// big-endian close code followed by the UTF-8 reason (RFC 6455 §5.5.1). A caller hands this
// straight to their WebSocket library's close/control-write call. The total is always <= 125
// bytes (2 code bytes + a reason capped at 123), so it is a valid control-frame payload.
func ControlPayload(err error, locale string) []byte {
	code, reason := Close(err, locale)
	payload := make([]byte, 2, 2+len(reason))
	payload[0] = byte(code >> 8)
	payload[1] = byte(code)
	return append(payload, reason...)
}

// coerce returns err as a *herr.Error, wrapping a non-herr error as a server fault so its
// detail stays server-side (available to logs) and never reaches the close reason.
func coerce(err error) *herr.Error {
	var he *herr.Error
	if errors.As(err, &he) {
		return he
	}
	return herr.New("INTERNAL").Kind(herr.KindInternal).Wrap(err)
}

// publicMessage extracts the safe, localized public message from the error by rendering its
// wire body (the same allow-listed DTO every transport uses) and reading the message field.
// Going through Body keeps wserr on the safe side of the public/internal split.
func publicMessage(he *herr.Error, locale string) string {
	raw, err := json.Marshal(he.Body(locale))
	if err != nil {
		return ""
	}
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &body)
	return body.Message
}

// capReason truncates a reason to maxReasonBytes, stepping back to a UTF-8 rune boundary so
// the close frame never carries a half-rune. Short reasons pass through untouched.
func capReason(s string) string {
	if len(s) <= maxReasonBytes {
		return s
	}
	cut := maxReasonBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
