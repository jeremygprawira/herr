package herr

// Public is, by definition, the COMPLETE set of fields that may cross the wire.
//
// This single-container design is the heart of the safe split: to answer "what can the
// client see?" you look at exactly one place. Everything NOT in Public (internal
// message, internal fields, cause, stack) is logs-only by construction.
//
// All fields are optional. Title/Message are human-readable display text; Metadata is a
// free-form bag for anything else the team wants the client to have (a support URL, an
// incident id, a doc link, ...). Because Metadata is public, only safe values belong in
// it — internal context goes through Error.With instead.
type Public struct {
	// Title is an optional heading (e.g. "Unable to connect your account").
	Title string
	// Message is the main user-facing sentence — the everyday field.
	Message string
	// Metadata is open-ended, team-specific public data. nil when unused.
	Metadata map[string]any
}

// Msg is shorthand for a Public carrying only a Message, so the common single-sentence
// case stays a one-liner: herr.New("X").Public(herr.Msg("...")).
func Msg(s string) Public {
	return Public{Message: s}
}

// Public sets the entire public surface at once and returns the receiver for chaining.
// Use it from a catalog definition or when you want to set several public fields
// together; use Title/Message/Meta to set them individually.
func (e *Error) Public(p Public) *Error {
	if e == nil {
		return nil
	}
	e.public = p
	return e
}

// Title sets only the public Title, leaving the rest of the public surface untouched.
func (e *Error) Title(title string) *Error {
	if e == nil {
		return nil
	}
	e.public.Title = title
	return e
}

// Message sets only the public Message, leaving the rest of the public surface
// untouched.
func (e *Error) Message(msg string) *Error {
	if e == nil {
		return nil
	}
	e.public.Message = msg
	return e
}
