package herr

import "errors"

// Field errors describe WHICH inputs failed validation and why, in a shape a front end can
// render next to each offending field. They are a PUBLIC, front-end-facing channel — not a
// developer/log channel — so each entry carries only the safe triple {field, code,
// message}. Any rejected value or validator reason a developer wants to debug goes through
// the normal internal .With(...) channel on the child and is NEVER auto-promoted here
// (that would risk leaking PII / internals, breaking C2).
//
// Two authoring styles:
//   - herr.FieldError(field, code, message) builds a single field error (the building
//     block, also used by the multi-error interop), and
//   - parent.FieldError(field, code, message) appends a child to a parent and returns the
//     parent, so a handler collects several at once and renders one typed `errors[]`.

// maxFieldErrors bounds how many children a parent will render (H5): a buggy validator or a
// hostile payload can't make the response array grow without limit. Past the cap, further
// children are dropped and a single marker child records the truncation.
const maxFieldErrors = 100

// FieldError builds a single field error: a herr error whose PUBLIC surface is the triple
// {field, code, message}. Its Kind is Unprocessable (422) so a standalone field error still
// maps sensibly across transports. The message is localizable via the normal resolution
// chain (it is stored as the literal fallback, NOT pinned as an inline override), so a
// Localizer can translate it by the code-derived key.
func FieldError(field, code, message string) *Error {
	return &Error{
		code:   code,
		kind:   KindUnprocessable,
		field:  field,
		public: Public{Message: message},
	}
}

// FieldError appends a per-field child to this (parent) error and returns the PARENT, so
// calls chain to collect several field errors before the parent is returned to the
// transport. The parent renders the children as the wire `errors[]`. Honors the H5 cap.
func (e *Error) FieldError(field, code, message string) *Error {
	if e == nil {
		return nil
	}
	if len(e.children) >= maxFieldErrors {
		// Record the truncation exactly once (when first crossing the cap), then drop.
		if len(e.children) == maxFieldErrors {
			e.children = append(e.children, FieldError("", "_errors_truncated", ""))
		}
		return e
	}
	e.children = append(e.children, FieldError(field, code, message))
	return e
}

// wireFieldError is the safe, allow-listed shape of a single `errors[]` entry. Like
// wireError, it is built by hand from explicit public parts — so a child's internal fields,
// cause, or stack can never ride along into the array.
type wireFieldError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// fieldEntry renders one error's PUBLIC parts into a wire `errors[]` entry, localizing its
// message through the same chain the top-level message uses. Reading only {field, code,
// message} is what keeps an entry safe: a promoted child's internal fields/cause/stack have
// no path into the array.
func (e *Error) fieldEntry(locale string) wireFieldError {
	return wireFieldError{
		Field:   e.field,
		Code:    e.code,
		Message: e.resolveMessage(locale),
	}
}

// fieldErrors renders the wire `errors[]` from two sources:
//
//  1. children appended explicitly via .FieldError, and
//  2. *herr.Error children PROMOTED from a wrapped multi-error aggregate (errors.Join /
//     go-multierror) — so a handler can validate with any aggregator and still return a
//     clean per-field response.
//
// A NON-herr child in the aggregate is deliberately NOT promoted: arbitrary error strings
// must never reach the client (C2). Such children remain visible to logs through the
// wrapped cause. The combined list is H5-bounded. Returns nil when empty so the field is
// omitted entirely.
func (e *Error) fieldErrors(locale string) []wireFieldError {
	var out []wireFieldError

	// 1. explicit children
	for _, c := range e.children {
		out = append(out, c.fieldEntry(locale))
	}

	// 2. promoted *herr.Error children from a wrapped aggregate (one level)
	for _, child := range aggregateChildren(e.cause) {
		var he *Error
		if errors.As(child, &he) {
			out = append(out, he.fieldEntry(locale))
		}
	}

	if out == nil {
		return nil
	}
	// H5: bound the rendered array regardless of how children arrived.
	if len(out) > maxFieldErrors {
		out = out[:maxFieldErrors]
		out = append(out, wireFieldError{Code: "_errors_truncated"})
	}
	return out
}
