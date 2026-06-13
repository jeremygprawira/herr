package herr

import "sync/atomic"

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

// defaultOverrides stores a process-wide override of the floor messages behind an
// atomic.Pointer, mirroring the localizer holder: reads are lock-free on the hot render
// path and a (rare, init-time) SetDefaults cannot data-race with concurrent renders. nil
// means "no override installed" — the built-in set is used.
var defaultOverrides atomic.Pointer[map[Kind]string]

// SetDefaults installs a process-wide override of the built-in floor messages. Intended to
// be called once at startup. Only the Kinds present in m are overridden; any Kind absent
// from m keeps its built-in floor. Passing nil clears the override and restores the
// built-in set. The override is the FLOOR only — an explicit public message still wins.
//
// The map is defensively copied so a later mutation of the caller's map cannot race with or
// silently change rendering.
func SetDefaults(m map[Kind]string) {
	if m == nil {
		defaultOverrides.Store(nil)
		return
	}
	cp := make(map[Kind]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	defaultOverrides.Store(&cp)
}

// defaultMessage returns the floor message for a Kind. It consults, in order: an installed
// SetDefaults override for that Kind, then the built-in floor, then the generic last-resort
// sentence. It is consulted by the wire layer ONLY when no explicit public message is
// present, so a rendered error is never blank.
func defaultMessage(k Kind) string {
	if over := defaultOverrides.Load(); over != nil {
		if m, ok := (*over)[k]; ok {
			return m
		}
	}
	if m, ok := defaultMessages[k]; ok {
		return m
	}
	return genericFloor
}
