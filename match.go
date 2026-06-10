package herr

import "errors"

// Is integrates *Error with the standard errors.Is. Two herr errors are considered equal
// when they share the same Code — identity is the stable code, not pointer equality or
// internal detail. This lets `errors.Is(err, ErrNotFound.New())` work, and (because
// *Error implements Unwrap) the standard library walks the chain for us.
func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	var t *Error
	if errors.As(target, &t) {
		return t.code == e.code
	}
	return false
}

// Is reports whether err (anywhere in its wrapped chain) is a herr error of this class's
// Code. It is the ergonomic catalog-side check:
//
//	if ErrNotFound.Is(err) { ... }
//
// It uses errors.As internally, so it transparently sees through wrapping added by other
// packages (fmt.Errorf("%w"), etc.).
func (c *Class) Is(err error) bool {
	if c == nil {
		return false
	}
	var he *Error
	if errors.As(err, &he) {
		return he.code == c.Code
	}
	return false
}
