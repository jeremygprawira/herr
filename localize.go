package herr

import (
	"strings"
	"sync/atomic"
)

// Localizer is the tiny interface herr uses to translate public text. herr core depends
// ONLY on this interface — never on a concrete i18n library — which is what keeps it
// i18n-agnostic. Adapters for go-i18n / x/text / a plain map satisfy it in a few lines.
//
// Localize is asked for a (locale, key) and the template params (in case the underlying
// library does its own interpolation/pluralization). It returns the translated string and
// ok=false when there is no translation, so herr can fall back.
type Localizer interface {
	Localize(locale, key string, params map[string]any) (msg string, ok bool)
}

// localizerHolder stores the process-wide Localizer behind an atomic.Value so reads are
// lock-free on the hot render path and a (rare, init-time) SetLocalizer cannot data-race
// with concurrent renders. The bool wrapper lets us store a typed nil safely.
var localizer atomic.Pointer[localizerBox]

type localizerBox struct{ l Localizer }

// SetLocalizer installs the global Localizer. Intended to be called once at startup.
// Passing nil disables localization (herr then uses literal/floor messages).
func SetLocalizer(l Localizer) {
	localizer.Store(&localizerBox{l: l})
}

// currentLocalizer returns the installed Localizer, or nil if none/disabled.
func currentLocalizer() Localizer {
	if box := localizer.Load(); box != nil {
		return box.l
	}
	return nil
}

// messageKey derives the i18n key for an error's public message from its Code, e.g.
// "ACCOUNT_CONNECT_FAILED" → "errors.account_connect_failed.message". Centralizing the
// convention here keeps it consistent and overridable in one place.
func messageKey(code string) string {
	return "errors." + strings.ToLower(code) + ".message"
}

// MessageKey sets an EXPLICIT i18n key for this error's public message, overriding the
// key derived from Code. Use it to share one translation across several codes, or to map a
// code onto a key that doesn't follow the `errors.<code>.message` convention. Returns the
// receiver for chaining.
func (e *Error) MessageKey(key string) *Error {
	if e == nil {
		return nil
	}
	e.msgKey = key
	return e
}

// effectiveMessageKey is the i18n key the resolver actually asks the Localizer for: the
// explicit override when set, otherwise the key derived from Code. One place owns the
// "explicit beats derived" rule so every lookup path stays consistent.
func (e *Error) effectiveMessageKey() string {
	if e.msgKey != "" {
		return e.msgKey
	}
	return messageKey(e.code)
}

// resolveMessage applies the public-message resolution chain for a locale:
//
//  1. an INLINE call-site message wins (you wrote the exact words; you mean them);
//  2. else a Localizer translation of the derived key, if one exists for the locale;
//  3. else the literal catalog Message (params substituted);
//  4. else the built-in Kind floor — so the body is never blank, never a raw key.
//
// Only the literal paths run through the injection-safe substituter; a Localizer is
// trusted to handle its own interpolation via the params it receives.
func (e *Error) resolveMessage(locale string) string {
	// 1. inline override
	if e.msgInline && e.public.Message != "" {
		return substitute(e.public.Message, e.params)
	}
	// 2. translation by effective key (explicit override, else derived from Code)
	if l := currentLocalizer(); l != nil {
		if s, ok := l.Localize(locale, e.effectiveMessageKey(), e.params); ok {
			return s
		}
	}
	// 3. literal catalog message
	if e.public.Message != "" {
		return substitute(e.public.Message, e.params)
	}
	// 4. floor
	return defaultMessage(e.kind)
}
