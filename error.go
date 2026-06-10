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

	// public is the user-facing surface: the ONLY data that may be serialized to the
	// client. It is exported as a struct (herr.Public) but stored here unexported so the
	// wire DTO — not reflection over *Error — controls exactly what is emitted.
	public Public

	// cause is the underlying error this one wraps, if any. It powers errors.Unwrap
	// (and therefore errors.Is/As over the chain). It is INTERNAL — it is surfaced to
	// logs, never to the client.
	cause error
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
// transport layer). For now it surfaces the code so a log line is identifiable; as more
// internal detail is added (internal message, cause), this string grows to include it.
func (e *Error) Error() string {
	return e.code
}

// Unwrap returns the wrapped cause (or nil), making *Error participate in the standard
// errors.Is / errors.As chain walking. Returning the INTERNAL cause here is safe: it
// only exposes the chain to server-side code, never to the serialized response.
func (e *Error) Unwrap() error {
	return e.cause
}
