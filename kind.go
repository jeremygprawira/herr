package herr

// Kind is the closed, universal classification of an error.
//
// It exists so a caller can describe an error ONCE, semantically, and have the right
// transport codes (HTTP status, gRPC code, WebSocket close code) and retryability fall
// out by convention. Every default is overridable; Kind only supplies the default.
//
// The zero value is deliberately KindInternal: an error that was never classified is
// the most dangerous one, so the safe default is "server fault / 500", never a success
// or a vague 400.
type Kind uint8

const (
	// KindInternal is an unexpected server-side fault (the zero value). → 500.
	KindInternal Kind = iota
	// KindInvalid means the request was understood but is semantically wrong. → 400.
	KindInvalid
	// KindUnauthorized means the caller is not authenticated. → 401.
	KindUnauthorized
	// KindForbidden means the caller is authenticated but not allowed. → 403.
	KindForbidden
	// KindNotFound means the addressed resource does not exist. → 404.
	KindNotFound
	// KindConflict means the request conflicts with current state. → 409.
	KindConflict
	// KindRateLimited means the caller sent too many requests. → 429.
	KindRateLimited
	// KindTimeout means the operation exceeded its deadline. → 504.
	KindTimeout
	// KindUnavailable means a dependency / the service is temporarily down. → 503.
	KindUnavailable
)

// kindHTTP maps each Kind to its default HTTP status code. This is the single source of
// truth for the HTTP side of the convention; transports read it via HTTPStatus().
//
// A map (rather than a slice indexed by Kind) is used for readability; the set is tiny
// and this is not on a hot path that would care about the lookup cost.
var kindHTTP = map[Kind]int{
	KindInternal:     500,
	KindInvalid:      400,
	KindUnauthorized: 401,
	KindForbidden:    403,
	KindNotFound:     404,
	KindConflict:     409,
	KindRateLimited:  429,
	KindTimeout:      504,
	KindUnavailable:  503,
}

// Kind sets the error's classification and returns the receiver for chaining.
//
// Builder methods are written to be nil-safe (calling on a nil *Error returns nil) so a
// chain that started from a nil value never panics; this keeps call sites free of nil
// checks.
func (e *Error) Kind(k Kind) *Error {
	if e == nil {
		return nil
	}
	e.kind = k
	return e
}

// Status sets an EXPLICIT HTTP status override and returns the receiver for chaining.
// Use it only when the Kind-derived default is not what you want.
func (e *Error) Status(code int) *Error {
	if e == nil {
		return nil
	}
	e.httpStatus = code
	return e
}

// HTTPStatus resolves the effective HTTP status using the "convention with override"
// rule:
//
//  1. an explicit Status() override wins, else
//  2. the Kind-derived default, else
//  3. 500 as the final safe floor (should be unreachable for known Kinds).
//
// Transports call this to set the response status line.
func (e *Error) HTTPStatus() int {
	if e == nil {
		return 500
	}
	if e.httpStatus != 0 {
		return e.httpStatus
	}
	if s, ok := kindHTTP[e.kind]; ok {
		return s
	}
	return 500
}
