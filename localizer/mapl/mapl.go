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
// back to its own message chain. Placeholder substitution arrives in a later
// cycle.
func (l *Localizer) Localize(locale, key string, params map[string]any) (string, bool) {
	table, ok := l.tables[locale]
	if !ok {
		return "", false
	}
	tmpl, ok := table[key]
	if !ok {
		return "", false
	}
	return tmpl, true
}
