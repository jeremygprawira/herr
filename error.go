// Package herr produces errors that carry two surfaces at once:
//
//   - a PUBLIC surface  — safe, user-facing, localizable data that may cross the wire
//   - an INTERNAL surface — rich detail (cause, fields, stack) that goes ONLY to logs
//
// The boundary between them is structural: internal data lives in UNEXPORTED fields
// and is never serialized, so a developer cannot leak it by accident. See the design
// spec in docs/specs for the full rationale.
//
// This file defines the core Error type and its most basic constructors/accessors.
// Higher-level behavior (catalog, builder chain, wire serialization, i18n, transports)
// is layered on top in sibling files and sub-packages.
package herr

import (
	"strings"
	"time"
)

// Error is the single runtime type the whole library revolves around.
//
// Every exported method returns *Error (never a bare `error`) so calls can be chained
// fluently, e.g. herr.New("X").Status(404).Wrap(cause). The internal/log-only fields
// are intentionally UNEXPORTED: external code (and reflection-based encoders such as
// encoding/json) cannot reach them, which is the first line of defense for the
// public/internal split.
type Error struct {
	// code is the stable, machine-readable identifier for this error class.
	// It is the one piece of data a consumer is guaranteed to be able to switch on,
	// and it never changes silently across versions (see the stability contract).
	code string

	// kind classifies the error (NotFound, Invalid, Internal, ...). It drives the
	// DEFAULT transport codes (HTTP/gRPC/WS) and retryability, so callers usually set
	// Kind once instead of repeating status numbers. The zero value is KindInternal
	// (see Kind's iota ordering) — an unclassified error is treated as a server fault.
	kind Kind

	// httpStatus, when non-zero, is an EXPLICIT override of the Kind-derived HTTP
	// status. Zero means "unset — derive from Kind". Keeping 0 as the sentinel lets the
	// override be optional without an extra bool.
	httpStatus int

	// grpcCode / wsClose are EXPLICIT transport-code overrides. Their zero values
	// (GRPCOK / 0) are never valid for an error, so zero cleanly means "derive from Kind".
	grpcCode GRPCCode
	wsClose  int

	// retry is the explicit tri-state retryability claim (RetryUnset = no claim). The
	// effective value is resolved against Kind at render time; see resolveRetry.
	retry Retry

	// retryAfter, when > 0, is a suggested delay before retrying. It is rendered as whole
	// seconds (HTTP Retry-After / gRPC RetryInfo) and, when set, implies retryability.
	retryAfter time.Duration

	// public is the user-facing surface: the ONLY data that may be serialized to the
	// client. It is exported as a struct (herr.Public) but stored here unexported so the
	// wire DTO — not reflection over *Error — controls exactly what is emitted.
	public Public

	// pubMeta holds dynamic public metadata added at the call site via WithPublic. It is
	// merged with public.Metadata at render time. Kept separate (and lazily allocated)
	// so the static catalog Metadata is never mutated by per-request additions.
	pubMeta map[string]any

	// params fills {name} placeholders in the public Title/Message at render time.
	// Lazily allocated; see template.go for the injection-safe substitution.
	params map[string]any

	// internal is the developer-only message (logs/debugging). It is part of the
	// INTERNAL surface and never serialized to the client.
	internal string

	// fields are internal structured context (key/value) for logs. INTERNAL — never
	// serialized. Lazily grown via With.
	fields []Field

	// cause is the underlying error this one wraps, if any. It powers errors.Unwrap
	// (and therefore errors.Is/As over the chain). It is INTERNAL — it is surfaced to
	// logs, never to the client.
	cause error

	// traceID is a correlation id echoed to the client (as `traceId`) AND to logs, so a
	// user can quote it to support and an operator can find the exact request. It is one
	// of the few values that intentionally appears on BOTH surfaces. Set via Trace;
	// transports inject one when unset.
	traceID string
}

// Trace sets the correlation id and returns the receiver for chaining.
func (e *Error) Trace(id string) *Error {
	if e == nil {
		return nil
	}
	e.traceID = id
	return e
}

// TraceID returns the correlation id (empty if unset).
func (e *Error) TraceID() string {
	if e == nil {
		return ""
	}
	return e.traceID
}

// New constructs a fresh *Error from a Code.
//
// Code is the only thing the library truly requires; everything else (status codes,
// messages, metadata) is optional and can be added via the builder methods or derived
// later. This is the "inline" authoring style — no catalog entry needed.
func New(code string) *Error {
	return &Error{code: code}
}

// Code returns the stable, machine-readable error code.
//
// Consumers branch on this value; it is also what errors.Is uses to compare two herr
// errors for equality (see Is, added in a later step).
func (e *Error) Code() string {
	return e.code
}

// Error implements the standard `error` interface.
//
// The string it returns is DEVELOPER-facing — intended for logs and debugging, not for
// end users (the user-facing message lives on the public surface and is rendered by the
// transport layer). It surfaces the code, the internal message, and the wrapped cause so
// a single log line is self-explanatory. None of this string is ever sent to a client.
func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(e.code)
	if e.internal != "" {
		b.WriteString(": ")
		b.WriteString(e.internal)
	}
	if e.cause != nil {
		b.WriteString(" (cause: ")
		b.WriteString(e.cause.Error())
		b.WriteString(")")
	}
	return b.String()
}

// Unwrap returns the wrapped cause (or nil), making *Error participate in the standard
// errors.Is / errors.As chain walking. Returning the INTERNAL cause here is safe: it
// only exposes the chain to server-side code, never to the serialized response.
func (e *Error) Unwrap() error {
	return e.cause
}
