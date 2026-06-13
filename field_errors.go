package herr

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

// fieldErrors renders this error's children into the wire `errors[]`, localizing each
// child's message through the same resolution chain the top-level message uses. Returns nil
// when there are no children, so the field is omitted from the body entirely.
func (e *Error) fieldErrors(locale string) []wireFieldError {
	if len(e.children) == 0 {
		return nil
	}
	out := make([]wireFieldError, 0, len(e.children))
	for _, c := range e.children {
		out = append(out, wireFieldError{
			Field:   c.field,
			Code:    c.code,
			Message: c.resolveMessage(locale),
		})
	}
	return out
}
