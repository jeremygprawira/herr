package herr

import "fmt"

// Field is a single key/value of internal structured context, destined for logs only.
// It is intentionally tiny and visibility-free: INTERNAL fields use this type, while
// PUBLIC data uses the map on Error.pubMeta. Keeping the two in separate storage (rather
// than one list with a visibility flag) makes the safe split impossible to get wrong —
// there is simply no public path out of `fields`.
type Field struct {
	Key string
	Val any
}

// With attaches an INTERNAL structured field (logs only). It never crosses the wire.
//
// Use it freely for debugging context — query ids, upstream names, raw values — without
// any risk of leaking, because the wire DTO has no access to these fields.
func (e *Error) With(key string, val any) *Error {
	if e == nil {
		return nil
	}
	e.fields = append(e.fields, Field{Key: key, Val: val})
	return e
}

// WithPublic attaches PUBLIC metadata that WILL cross the wire (merged into the response
// `metadata` object). Only put safe, non-sensitive values here; for internal context use
// With instead.
func (e *Error) WithPublic(key string, val any) *Error {
	if e == nil {
		return nil
	}
	if e.pubMeta == nil {
		e.pubMeta = make(map[string]any)
	}
	e.pubMeta[key] = val
	return e
}

// Meta is an alias for WithPublic, provided because "metadata" reads naturally at call
// sites that think of this as "attach public metadata".
func (e *Error) Meta(key string, val any) *Error { return e.WithPublic(key, val) }

// Internal sets the developer-only message (logs/debugging). INTERNAL — never sent to a
// client. Overwrites any previous internal message.
func (e *Error) Internal(msg string) *Error {
	if e == nil {
		return nil
	}
	e.internal = msg
	return e
}

// Internalf is Internal with fmt-style formatting. NOTE: the format string is a constant
// you control; user-supplied data goes only in the args, never as the format itself.
func (e *Error) Internalf(format string, args ...any) *Error {
	if e == nil {
		return nil
	}
	e.internal = fmt.Sprintf(format, args...)
	return e
}

// Wrap records the underlying cause, enabling errors.Unwrap/Is/As to walk the chain. The
// cause is INTERNAL: it enriches logs but is never serialized to the client.
func (e *Error) Wrap(cause error) *Error {
	if e == nil {
		return nil
	}
	e.cause = cause
	return e
}
