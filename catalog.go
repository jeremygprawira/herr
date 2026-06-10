package herr

// Class is an immutable catalog template for an error.
//
// You declare a Class once (typically as a package-level var via Define) and stamp out
// fresh *Error values from it with New(). Centralizing definitions gives you a single
// source of truth for an error's code, classification, transport overrides, and default
// public surface — handy for consistency, auditing, and i18n.
//
// IMMUTABILITY CONTRACT (this is what makes C1 hold): a Class is treated as read-only
// after Define. New() copies its scalar fields into a fresh *Error and SHARES the
// Public.Metadata map by reference — but that map is only ever READ during rendering
// (per-request public metadata is written to a separate map via WithPublic). Because no
// code path mutates Class state, many goroutines can New()/build/render from one Class
// concurrently with no locking and no cross-request contamination.
type Class struct {
	// Code is the stable, machine-readable identifier (required).
	Code string
	// Kind classifies the error and supplies default transport codes.
	Kind Kind
	// HTTP optionally overrides the Kind-derived HTTP status (0 = derive from Kind).
	HTTP int
	// Public is the default user-facing surface stamped onto every New() instance.
	Public Public
}

// Define registers an error class and returns an immutable handle to it.
//
// It copies the provided Class by value so the caller's literal cannot later be mutated
// to affect already-defined errors. The returned *Class is meant to be stored in a
// package-level var:
//
//	var ErrNotFound = herr.Define(herr.Class{Code: "NOT_FOUND", Kind: herr.KindNotFound})
func Define(c Class) *Class {
	cc := c // defensive copy: decouple from the caller's struct literal
	return &cc
}

// New stamps out a fresh *Error pre-filled with this class's defaults.
//
// The new instance gets its OWN identity for everything mutable per request: internal
// message, internal fields, dynamic public metadata, cause, trace id, and stack all start
// empty and are populated via the builder. The class's scalar fields are copied; its
// Public struct is copied by value (the Metadata map inside is shared read-only — see the
// immutability contract on Class). The result: instances are fully isolated from each
// other and from the class.
func (c *Class) New() *Error {
	return &Error{
		code:       c.Code,
		kind:       c.Kind,
		httpStatus: c.HTTP,
		public:     c.Public,
	}
}
