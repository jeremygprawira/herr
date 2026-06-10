# herr — Design Spec

**Status:** Draft for review
**Date:** 2026-06-11
**Language:** Go 1.26+

`herr` ("handle error") is a framework-, logger-, and i18n-library-agnostic Go
package for producing errors that are simultaneously **safe** (nothing internal
leaks to the client), **user-friendly** (a render-ready, localized, well-written
message), and **developer-friendly** (full internal detail + structured fields go
to logs). It works across REST, gRPC, and WebSocket, and across the repository,
usecase, and handler layers.

---

## 1. Goals & Non-Goals

### Goals
- One error type that carries a **public surface** (shown to users) and an
  **internal surface** (logs only), with the boundary enforced structurally.
- **Flexibility first:** `Code` is the only required field; everything else is
  optional and auto-derived, and every default is overridable.
- Encode the Wix "anatomy of a good error message" (what happened / why /
  reassurance / way out / help) as optional, structured, render-ready data.
- Be agnostic: core imports **zero** third-party logging or i18n libraries.
- Ship REST, gRPC, and WebSocket adapters in v1.

### Non-Goals
- Not a translation engine (plurals/grammar belong to the i18n lib behind the
  `Localizer` interface).
- Not a copywriting tool — it **enforces** message quality, it does not **write**
  brand voice.
- Not a UI toolkit — the API carries presentation **semantics**, not pixels.

---

## 2. Design Principles

1. **Flexibility first.** Only `Code` is mandatory. Minimal usage must be trivial.
2. **Convention with override.** Every field auto-derives a sensible default; any
   default can be overridden or disabled. Magic is always a fallback, never a trap.
3. **Default-safe.** Anything not explicitly public is internal. Unhandled / non-herr
   errors render as a generic safe message — never blank, never leaking.
4. **One boundary.** Everything the client can see lives in one place (`Public`);
   everything else is internal by construction.
5. **Agnostic core.** No logging/i18n/framework deps in `herr` core; integrations are
   optional sub-packages behind tiny interfaces.

---

## 3. Package Layout

```
herr/                       # core — ZERO third-party deps
  herr/httperr              # REST (net/http): middleware + Write()
  herr/grpcerr              # gRPC: interceptors + status mapping
  herr/wserr                # WebSocket: close-code mapping
  herr/adapter/{zap,logrus,zerolog,slog}   # optional logger adapters (~10 lines each)
  herr/localizer/{mapl,goi18n,xtext}       # optional i18n adapters
  cmd/herrlint              # static analyzer (fast-follow)
```
Everything outside `herr/` core is opt-in; import only what you use.

---

## 4. Core Types

```go
// Class is an immutable catalog template. Never mutated after Define().
type Class struct {
    Code      string        // REQUIRED, stable, machine-readable
    Kind      Kind          // category; drives default HTTP/gRPC/WS/severity
    HTTP      int           // optional override of Kind default
    GRPC      codes.Code    // optional override
    WS        int           // optional override (close code)
    Retryable bool          // semantic; derives a retry action + Retry-After
    Public    Public        // everything the user may see (all optional)
}

// Public is, by definition, the COMPLETE set of fields that can cross the wire.
type Public struct {
    Title       string    // "what happened"           (optional)
    Message     string    // main text / "why"          (optional; floor fallback)
    Reassurance string    // "what's safe"              (optional)
    Support     *Support  // "way out / help"           (optional)
    Actions     []Action  // derived from Retryable/Support if omitted (optional)
}

type Support struct { Label, URL string }

type Action struct {
    Type  ActionType // retry | link | dismiss | navigate | support
    Label string
    URL   string     // for link/navigate
}

// Error is the single runtime type. Internal fields are UNEXPORTED so external
// reflection/marshaling cannot reach them.
type Error struct {
    class    *Class            // read-only reference for defaults
    code     string
    // public (lazily set)
    public   Public
    params   map[string]any    // template params; nil until used
    pubFields []Field          // explicitly-public structured fields; nil until used
    // internal — logs only, never serialized to client
    internal  string
    fields    []Field          // internal structured context; nil until used
    cause     error
    stack     []uintptr        // captured ONLY for server/Internal kinds
    traceID   string
}

type Field struct { Key string; Val any; Visible Visibility } // Internal (default) | Public
```

`*Error` implements `error`, `Unwrap() error`, and is `errors.Is`/`errors.As`
compatible (`Is` matches by `Code`).

---

## 5. Authoring: Two Styles, One Type

```go
// A) Catalog (recommended for shared/domain errors)
var ErrAccountConnect = herr.Define(herr.Class{
    Code:      "ACCOUNT_CONNECT_FAILED",
    Kind:      herr.KindUnavailable,           // "our end" → server-owned, retryable
    HTTP:      502,
    Retryable: true,
    Public: herr.Public{
        Message: "We couldn't connect your account due to a technical issue on our end. Please try again.",
        Support: &herr.Support{Label: "Customer Care", URL: "https://amartha.com/support"},
    },
})
return ErrAccountConnect.New().Reassure("Your changes were saved.").With("upstream", "kyc-svc").Wrap(err)

// B) Inline (no catalog required)
return herr.New("ACCOUNT_CONNECT_FAILED").
    Kind(herr.KindUnavailable).
    Public(herr.Msg("We couldn't connect your account due to a technical issue on our end.")).
    Wrap(err)
```

`herr.Msg(s)` is shorthand for `Public{Message: s}` so the common case is a one-liner.

### Builder surface (all return `*Error`, chainable, nil-safe)
`.Kind` `.Status` `.GRPC` `.WS` `.Public` `.Title` `.Message` `.Reassure`
`.Support` `.Action` `.Param/.Params` `.With` (internal field) `.WithPublic`
(public field) `.Internal/.Internalf` `.Trace` `.WithStack` `.Wrap`.

---

## 6. Convention-with-Override Defaults

| Field | Auto-derived from | Override |
|---|---|---|
| `HTTP` / `GRPC` / `WS` | `Kind` | set explicitly |
| `display.severity` | `Kind` | set explicitly |
| i18n keys | `Code` → `errors.<code>.{title,message,reassurance}` | set explicit key / disable |
| `actions` | `Retryable` + `Support` | explicit `Actions[]` |
| public message | resolution chain (§7) | inline `.Public()` |

Writing just `Code` yields a working, render-ready, localized-if-available error.

---

## 7. Message Resolution (locale `L`)

1. Inline `.Public()/.Message()` at call site → wins.
2. Explicit i18n key + configured `Localizer` resolves for `L` → translated.
3. No key set → **magic**: derive `errors.<code>.message`; if `Localizer` resolves → translated. If not, fall through (never shows a raw key).
4. `Public.Message` literal (params filled) → shown.
5. Built-in default for `Kind`.
6. Generic safe floor (`"Something went wrong."` + code + traceId).

Title and Reassurance follow the same chain with their own derived keys. **Only
public text is localized; internal/log text stays one language.**

---

## 8. Built-in Default Messages

~10 calm, Wix-principled fallbacks keyed by `Kind` (own-it for 5xx, no blame,
actionable, support path), routed through the same `Localizer` so they are
translatable. Globally overridable via `herr.SetDefaults(...)`. Used only as the
floor — never override an explicit message.

---

## 9. Render-Ready `display` (semantic-only)

The response carries a `display` object with **semantics**, never presentation:

```json
{
  "code": "ACCOUNT_CONNECT_FAILED",
  "traceId": "f1c2-...",
  "display": {
    "severity": "error",
    "title": "Unable to connect your account",
    "message": "We couldn't connect your account due to a technical issue on our end.",
    "reassurance": "Your changes were saved.",
    "actions": [
      { "type": "retry",   "label": "Try Again" },
      { "type": "support", "label": "Contact Customer Care", "url": "https://amartha.com/support" }
    ]
  }
}
```

- **No** `style` (modal/toast) and **no** button hierarchy on the wire — that is the
  client's decision (fixes backend-driven-UI coupling).
- Reference renderers (`@herr/react`, Flutter `HerrErrorView`) map
  `severity → presentation` and `action.type → behavior/emphasis`, fully
  overridable. Shipped as **fast-follow**.
- Clients may instead consume `code` + `params` and localize/render fully
  client-side (dual mode — avoids double-localization conflicts).

---

## 10. The Safe Split (Security Model)

- `Public` is the **only** thing that crosses the wire. Internal data
  (`internal`, internal `fields`, `cause`, `stack`) lives in **unexported** fields.
- **Whitelist serialization:** a single `(e *Error) wire(locale) wireError` builds an
  explicit allowlisted DTO. `MarshalJSON` delegates to it, so even an accidental
  `json.Marshal(err)` is safe. No reflection over the full struct, ever.
- Non-`herr` errors reaching a transport boundary are coerced to
  `code:"INTERNAL"` + generic safe message + traceId.

---

## 11. Logging — Three Modes (core imports no logger)

- **Pull** (always available): `herr.Attrs(err) []slog.Attr`,
  `herr.LogFields(err) map[string]any`, `herr.LogRecord(err)`.
- **Auto** (opt-in): pass a one-method `Logger` adapter to transport middleware;
  herr logs unexpected errors at the boundary. Adapters: zap/logrus/zerolog/slog.
- **None / BYO:** configure nothing.

**Default policy:** auto-log fires for `Internal`/5xx only. 4xx is **not** logged by
default (configurable, sampleable) — prevents attacker-driven 4xx log floods.

---

## 12. Transports (all v1)

- **httperr:** `Middleware`, `Write(w, r, err)`, JSON body, status from error,
  locale from `Accept-Language` (allowlist-matched, §14).
- **grpcerr:** unary + stream interceptors → `status.Status` with whitelisted
  details; locale from metadata.
- **wserr:** error → close code + public reason (≤123 bytes, UTF-8 safe);
  helper to send the control frame; locale from handshake.

---

## 13. Data Flow (repository → usecase → handler)

- **Repository** translates infra errors into `herr` at the boundary
  (`sql.ErrNoRows → ErrNotFound.New().Wrap(err)`); attaches low-level internal detail.
- **Usecase** adds domain context via `.With(...)`, may re-map `Kind`, otherwise lets
  it bubble. Wrapping preserves the chain; fields accumulate.
- **Handler** builds nothing — one call: `httperr.Write(w, r, err)`. The adapter
  resolves locale, renders the safe body, sets status, and (if configured) auto-logs.

---

## 14. Hardening Requirements (Critical / High fixes — first-class)

| # | Requirement |
|---|---|
| **C1** | `Class` immutable; **all** per-request state lives on the instance, lazily allocated. Builder never writes through to `Class`. Verified by a `-race` concurrency test hammering one catalog entry. |
| **C2** | Whitelist serialization via explicit `wireError` DTO; internal fields unexported; `MarshalJSON` emits only the safe DTO. Verified by a fuzz test that injects secret markers into every internal field and asserts they never appear in any render. |
| **H1** | `display` is semantic-only (no `style`/button hierarchy). Presentation lives in client renderers. |
| **H2** | Ship `herr.ErrInvalidCredentials` (coarse). Precise auth reason → logs only. Lint warns on multiple identity/auth-specific codes. |
| **H3** | Named-placeholder `{param}` substitution only — never `Sprintf` with user input as format. Missing param → empty (prod) / visible + warn (StrictMode), never panic. Ship `herr.MaskEmail/MaskPhone`. Lint flags PII-shaped params in public text. |
| **H4** | Locale resolved against a configured supported-locale allowlist (`SetSupportedLocales`) via `x/text/language`; caches keyed on resolved locale, never raw header; cap parsed tags. |
| **H5** | Cheap error path: conditional stack capture (server/Internal only, or `.WithStack()`); lazy alloc of `params`/`fields`/`actions`; no-4xx-logging default; bounds on field count (≤64), validation entries (≤100), and string length, with truncation markers. |

### Global-safety riders
- `traceID` is random (UUID / propagated OTel id), never sequential.
- Global config (`SetLocalizer/SetDefaults/SetLogger/SetSupportedLocales`) is
  init-time only, backed by `atomic.Value`.

---

## 15. Message-Quality Layer

- **Cookbook (docs):** the Wix principles as a checklist + good/bad pairs; built-ins
  as worked examples. Ships in v1.
- **`herr.StrictMode()` (v1):** runtime/test validation — empty public message on a
  user-facing kind, `public == internal`, unfilled placeholders, missing
  support/way-out on 5xx, PII params, length bounds. Fails tests so bad messages
  can't merge.
- **`herrlint` (fast-follow):** `go vet`-style static analyzer for the same rules at
  CI time, plus the enumeration-oracle heuristic (H2).
- Honest caveat (documented): these catch leaks/blame/jargon/missing-pieces, not
  *tone* — tone stays a human review job.

---

## 16. Testing Strategy

- Table tests: builder, wrapping/`Is`/`As`, resolution chain, defaults.
- **Race test (C1):** concurrent use of a single catalog entry under `-race`.
- **Leak fuzz (C2):** secret markers in internal fields never appear in any render.
- Golden tests per transport (REST/gRPC/WS) + locale tests.
- Auto-log capture tests; bounds/truncation tests; DoS-shape tests (mass 4xx → no
  stack, no log, bounded output).

---

## 17. Stability Contract

`Code` values and the response JSON schema (including `display.actions[].type`) are
the public API, governed by semver. Public **message text** may change freely;
`Code` values never silently change. Clients must ignore unknown `action.type`.

---

## 18. Scope

**v1:** core (`herr`) + httperr + grpcerr + wserr; `Localizer` interface + `mapl`
adapter; logger `Logger` interface + zap/logrus/zerolog/slog adapters; built-in
defaults + cookbook + `StrictMode`; all §14 hardening.

**Fast-follow (own specs):** `cmd/herrlint`; reference renderers `@herr/react` and
Flutter `HerrErrorView`; `goi18n`/`xtext` localizer adapters.

---

## 19. Open Questions

- Should `display` be emitted always, or gated by an `Accept` / opt-in header so
  pure service-to-service consumers get a leaner body?
- Default supported-locale set out of the box (e.g. `en` only) vs require explicit
  `SetSupportedLocales`?
- Is `navigate` action in-scope for v1 or fast-follow with the renderers?
