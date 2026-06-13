package herr

import (
	"fmt"
	"strings"
)

// Param sets a single named template parameter and returns the receiver for chaining.
// Parameters fill {name} placeholders in the public Title/Message at render time.
func (e *Error) Param(name string, value any) *Error {
	if e == nil {
		return nil
	}
	if e.params == nil {
		e.params = make(map[string]any)
	}
	e.params[name] = value
	return e
}

// Params sets several template parameters at once.
func (e *Error) Params(m map[string]any) *Error {
	if e == nil {
		return nil
	}
	for k, v := range m {
		e.Param(k, v)
	}
	return e
}

// substitute replaces {name} placeholders in tmpl with values from params.
//
// SECURITY (H3): this is a deliberate hand-written, SINGLE-PASS scanner — NOT
// fmt.Sprintf. Two consequences matter:
//
//   - A parameter value is inserted as literal text, so format verbs like %s or %d in a
//     value are never interpreted (no format-string injection).
//   - Substituted text is never re-scanned, so a value that itself contains "{evil}"
//     cannot trigger another lookup. Placeholders only ever come from the trusted
//     template, never from (potentially attacker-controlled) values.
//
// An unfilled placeholder collapses to empty rather than exposing raw braces to a user.
// A "{" with no matching "}" is emitted verbatim.
func substitute(tmpl string, params map[string]any) string {
	if tmpl == "" || !strings.ContainsRune(tmpl, '{') {
		return tmpl // fast path: nothing to do
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
		// Found '{' — find the matching '}'.
		end := strings.IndexByte(tmpl[i:], '}')
		if end < 0 {
			// No closing brace: emit the rest verbatim and stop.
			b.WriteString(tmpl[i:])
			break
		}
		name := tmpl[i+1 : i+end] // text between the braces
		if v, ok := params[name]; ok {
			b.WriteString(fmt.Sprint(v)) // value inserted as text, never re-scanned
		} else if strictModeOn() {
			// StrictMode (H3): leave the unfilled placeholder VISIBLE so a developer sees
			// the missing Param immediately — production collapses it to empty instead.
			b.WriteString(tmpl[i : i+end+1])
		}
		// (missing param in production → write nothing → placeholder collapses to empty)
		i += end + 1 // advance past the closing '}'
	}
	return b.String()
}
