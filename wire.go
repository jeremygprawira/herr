package herr

import "encoding/json"

// wireError is the EXPLICIT allow-list of everything that may be serialized to a client.
//
// This type is the linchpin of the C2 security guarantee: the response body is built by
// hand from a fixed set of fields, never by reflecting over *Error. Adding a field to
// *Error therefore cannot accidentally leak it — a field only crosses the wire if it is
// added here on purpose. Internal data (internal message, fields, cause, stack) has no
// representation in this struct and so can never appear in output.
//
// `omitempty` keeps the body lean: unset optional fields disappear entirely rather than
// rendering as empty strings/objects.
type wireError struct {
	Code     string         `json:"code"`
	Title    string         `json:"title,omitempty"`
	Message  string         `json:"message,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// wire builds the safe DTO for a given locale.
//
// It is the SINGLE place that decides what the client sees, so all rendering paths
// (MarshalJSON here, and every transport adapter later) funnel through it and inherit
// the same guarantees. The locale parameter will drive message localization in a later
// step; for now the literal public fields are used as-is.
func (e *Error) wire(locale string) wireError {
	if e == nil {
		// Defensive: a nil error still produces a safe, generic body rather than
		// panicking or emitting null.
		return wireError{Code: "INTERNAL"}
	}
	return wireError{
		Code:     e.code,
		Title:    e.public.Title,
		Message:  e.public.Message,
		Metadata: e.public.Metadata,
	}
}

// MarshalJSON makes *Error safe to hand to encoding/json directly.
//
// Crucially it DELEGATES to wire(), so even an accidental json.Marshal(err) somewhere in
// a codebase emits only the allow-listed public surface — defense in depth on top of the
// unexported internal fields. Transports normally call wire() explicitly with the
// request locale; this method is the safety net for everything else.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.wire(""))
}
