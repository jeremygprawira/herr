package herr

import "time"

// Retry is a TRI-STATE retryability signal. The whole point of three states (rather than
// a bool) is that "the developer never said" must be distinguishable from "the developer
// said no". The zero value is RetryUnset — i.e. "no claim" — so forgetting to set it can
// never be misread as an assertion that the error is not retryable.
type Retry uint8

const (
	// RetryUnset is the zero value: no explicit claim. The effective value is then
	// derived from Kind, and if Kind has no opinion the signal is omitted from the wire.
	RetryUnset Retry = iota
	// RetryYes asserts the operation may be retried.
	RetryYes
	// RetryNo asserts the operation must NOT be retried.
	RetryNo
)

// kindRetry maps Kinds that have a clear retryability stance. Kinds NOT present here
// (notably KindInternal) are intentionally "unknown" — a 500 might be transient or
// permanent, so herr makes no claim rather than guessing.
var kindRetry = map[Kind]Retry{
	KindUnavailable: RetryYes, // dependency temporarily down → try again
	KindTimeout:     RetryYes, // deadline exceeded → try again
	KindRateLimited: RetryYes, // slow down, then retry (see Retry-After)
	KindInvalid:     RetryNo,  // request is wrong; retrying won't help
	KindUnauthorized: RetryNo,
	KindForbidden:    RetryNo,
	KindNotFound:     RetryNo,
	KindConflict:     RetryNo,
}

// Retry sets an explicit retryability claim and returns the receiver for chaining. An
// explicit value always wins over the Kind-derived default.
func (e *Error) Retry(r Retry) *Error {
	if e == nil {
		return nil
	}
	e.retry = r
	return e
}

// RetryAfter sets a suggested delay before retrying and returns the receiver for
// chaining. Setting a delay implies the error is retryable (see resolveRetry).
func (e *Error) RetryAfter(d time.Duration) *Error {
	if e == nil {
		return nil
	}
	e.retryAfter = d
	return e
}

// resolveRetry computes the EFFECTIVE tri-state using convention-with-override:
//  1. an explicit Retry() claim wins, else
//  2. a positive RetryAfter implies RetryYes, else
//  3. the Kind-derived stance, else
//  4. RetryUnset ("unknown") — which the wire layer renders by OMITTING the field.
func (e *Error) resolveRetry() Retry {
	if e == nil {
		return RetryUnset
	}
	if e.retry != RetryUnset {
		return e.retry
	}
	if e.retryAfter > 0 {
		return RetryYes
	}
	if r, ok := kindRetry[e.kind]; ok {
		return r
	}
	return RetryUnset
}
