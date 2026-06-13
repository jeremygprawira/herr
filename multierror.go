package herr

// herr is multi-error-AGNOSTIC the same way it is logger- and i18n-agnostic: it speaks the
// SHAPE of an aggregate error through a duck-typed interface check, never importing a
// concrete multi-error library. A developer keeps using whatever aggregator they like —
// stdlib errors.Join or hashicorp/go-multierror — and herr flattens it all the same.

// aggregateChildren returns the immediate child errors of a multi-error aggregate, or nil
// when err is not an aggregate. It recognizes both common shapes STRUCTURALLY, with no
// import of either library:
//
//   - interface{ Unwrap() []error }        — stdlib errors.Join (Go 1.20+)
//   - interface{ WrappedErrors() []error } — hashicorp/go-multierror
//
// Keeping this a tiny structural check is what preserves the zero-dependency core while
// still interoperating with the ecosystem's aggregators.
func aggregateChildren(err error) []error {
	if err == nil {
		return nil
	}
	if a, ok := err.(interface{ Unwrap() []error }); ok {
		return a.Unwrap()
	}
	if a, ok := err.(interface{ WrappedErrors() []error }); ok {
		return a.WrappedErrors()
	}
	return nil
}
