package herr

import (
	"crypto/rand"
	"encoding/hex"
)

// NewTraceID mints a fresh, random correlation id.
//
// It exists so a transport can stamp an id onto an error that arrived without one — giving
// the client something to quote to support and the operator something to grep for, even
// when the caller never set a trace via Trace. It is deliberately a free function (not tied
// to an *Error) so transports can generate an id up front and apply it where appropriate.
//
// The id is 16 cryptographically-random bytes, hex-encoded to 32 lowercase characters. That
// is wide enough that two concurrent requests effectively never collide, while staying a
// plain, log-safe ASCII token. crypto/rand never fails on supported platforms, so the id is
// always full-length and never empty.
func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
