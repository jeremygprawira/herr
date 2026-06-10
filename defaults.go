package herr

// defaultMessages is the built-in FLOOR: a calm, well-written public sentence per Kind,
// used only when the developer supplied no explicit message. They follow the message
// principles this library is built around — own the failure for server faults, no blame,
// stay actionable, and never expose internals. They are English; the i18n layer (added
// later) can translate them by key, and SetDefaults can override the whole set.
var defaultMessages = map[Kind]string{
	KindInternal:     "Something went wrong on our end. Please try again. If it keeps happening, contact support.",
	KindInvalid:      "Some of the information provided isn't valid.",
	KindUnauthorized: "You need to sign in to continue.",
	KindForbidden:    "You don't have access to this.",
	KindNotFound:     "We couldn't find what you're looking for.",
	KindConflict:     "This conflicts with the current state. Please refresh and try again.",
	KindRateLimited:  "You've made too many requests. Please slow down and try again shortly.",
	KindTimeout:      "This took too long to complete. Please try again.",
	KindUnavailable:  "We're temporarily unable to handle this. Please try again in a moment.",
}

// genericFloor is the last-resort message when even the Kind is unrecognized. It must
// never be empty and must never leak — a deliberately vague, safe sentence.
const genericFloor = "Something went wrong. Please try again."

// defaultMessage returns the floor message for a Kind, falling back to the generic floor.
// It is consulted by the wire layer ONLY when no explicit public message is present, so a
// rendered error is never blank.
func defaultMessage(k Kind) string {
	if m, ok := defaultMessages[k]; ok {
		return m
	}
	return genericFloor
}
