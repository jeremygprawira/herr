// Package mapl provides a map-backed implementation of the herr.Localizer
// interface. It translates an error's public message by looking a (locale, key)
// pair up in an in-memory table — the simplest possible localizer, ideal for
// small apps, tests, or embedding translations as Go literals.
//
// The herr core asks a Localizer for keys like "errors.<code>.message" (see the
// core's messageKey). This package stores those templates per locale and fills
// their {name} placeholders from the params the core passes through. It depends
// only on the standard library, keeping the root module third-party-free.
package mapl

import (
	"fmt"
	"strings"
)

// Localizer is a map-backed herr.Localizer. The outer map is keyed by locale
// (e.g. "en", "id") and each inner map maps an i18n key to its template string.
// It is read-only after construction, so concurrent renders need no locking.
type Localizer struct {
	tables map[string]map[string]string
}

// New builds a Localizer from a locale→(key→template) table.
//
// The constructor will defensively copy the input in a later cycle; for now it
// stores the table directly so the first lookup test can pass.
func New(tables map[string]map[string]string) *Localizer {
	return &Localizer{tables: tables}
}

// Localize implements herr.Localizer. It looks up the template stored for
// (locale, key); if none exists it returns ("", false) so the herr core falls
// back to its own message chain. When a template is found, its {name}
// placeholders are filled from params via the injection-safe substituter below.
func (l *Localizer) Localize(locale, key string, params map[string]any) (string, bool) {
	table, ok := l.tables[locale]
	if !ok {
		return "", false
	}
	tmpl, ok := table[key]
	if !ok {
		return "", false
	}
	return substitute(tmpl, params), true
}

// substitute replaces {name} placeholders in tmpl with values from params. It is
// a deliberate single-pass, hand-written scanner that mirrors the herr core's
// own substituter exactly, so a translated message behaves identically to a
// literal one.
//
// SECURITY: this is NOT fmt.Sprintf-with-tmpl-as-format. Two properties matter:
//   - A param value is inserted as literal text, so format verbs (%s, %d) inside
//     a value are never interpreted (no format-string injection).
//   - Substituted text is never re-scanned, so a value that itself contains
//     "{evil}" cannot trigger another lookup. Placeholders only ever come from
//     the trusted template, never from (possibly attacker-controlled) values.
//
// An unfilled placeholder collapses to empty rather than leaking raw braces; a
// "{" with no matching "}" is emitted verbatim.
func substitute(tmpl string, params map[string]any) string {
	if tmpl == "" || !strings.ContainsRune(tmpl, '{') {
		return tmpl // fast path: nothing to substitute
	}
	var b strings.Builder
	b.Grow(len(tmpl))
	for i := 0; i < len(tmpl); {
		c := tmpl[i]
		if c != '{' {
			b.WriteByte(c)
			i++
			continue
		}
		// Found '{' — locate the matching '}'.
		end := strings.IndexByte(tmpl[i:], '}')
		if end < 0 {
			// No closing brace: emit the rest verbatim and stop.
			b.WriteString(tmpl[i:])
			break
		}
		name := tmpl[i+1 : i+end] // text between the braces
		if v, ok := params[name]; ok {
			b.WriteString(fmt.Sprint(v)) // inserted as text, never re-scanned
		}
		// (missing param → write nothing → placeholder collapses to empty)
		i += end + 1 // advance past the closing '}'
	}
	return b.String()
}
