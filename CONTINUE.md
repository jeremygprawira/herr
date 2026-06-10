# herr — Implementation Progress & Handoff

This file is the **single source of truth for where implementation stands**. Update
it after every meaningful step so work can resume in a fresh session without context
loss. If you are picking this up cold: read this file, then the spec, then run the
tests, then continue from the first unchecked box.

- **Spec:** `docs/specs/2026-06-11-herr-design.md` (the design is final — no open questions)
- **Module:** `github.com/jeremygeraldprawira/herr`
- **Go:** 1.26.1
- **Method:** strict TDD (one test → one impl → repeat). Every `.go` file is
  thoroughly commented to explain the *flow* of each func/process.

## How to resume
```bash
cd ~/Documents/herr
go test ./...        # everything should be green
git log --oneline    # see what's done
```
Then open this file and start at the first unchecked `[ ]`.

## Conventions (do not drift)
- **TDD vertical slices.** Write ONE failing test, make it pass, refactor, commit. Never
  write a batch of tests ahead of implementation.
- **Comments on every file:** package doc comment + per-type + per-func explaining the
  flow. Aim for "a junior can follow the process by reading top to bottom."
- **Security gates first.** C1 (no shared mutable state) and C2 (whitelist
  serialization, no leaks) get tests *before* the surface grows.
- **Commit after each green cycle** with a clear message.

## Build order (check off as completed)

### Phase 0 — setup
- [x] `go mod init`
- [x] spec finalized
- [x] this handoff doc

### Phase 1 — core (`herr` package)  ← CURRENT
- [ ] `Kind` enum + default HTTP/gRPC/WS + retryability mapping
- [ ] `Error` type: `New(code)`, `Error()`, `Code()`, `Kind()`
- [ ] `Public` struct {Title, Message, Metadata} + `Msg()` shorthand
- [ ] builder: `.Status/.GRPC/.WS/.Kind/.Title/.Message/.Public/.Meta/.WithPublic/.With/.Internal/.Param/.Trace/.WithStack/.Wrap`
- [ ] **C2 leak gate:** `wire()` DTO + `MarshalJSON` emits ONLY allowlist; fuzz test injecting secrets into internal fields
- [ ] wrapping: `.Wrap`, `Unwrap`, `errors.Is`/`As` by code
- [ ] catalog: `Define(Class)` + `.New()` clone
- [ ] **C1 race gate:** concurrency test hammering one catalog entry under `-race`
- [ ] `Retry` tri-state → wire omits when unknown; `Retryable`/`RetryAfter`
- [ ] message resolution chain (§7) + `Localizer` interface
- [ ] built-in default messages (en) + `SetDefaults`
- [ ] lazy alloc + bounds (H5); conditional stack capture
- [ ] logging pull API: `Attrs/LogFields/LogRecord`; `Logger` interface

### Phase 2 — i18n
- [ ] `SetSupportedLocales` + `x/text/language` matching (H4)
- [ ] `localizer/mapl` (map-based)
- [ ] embedded `en` + `id` floor-message bundles

### Phase 3 — transports
- [ ] `httperr`: Middleware + Write + Retry-After + Accept-Language
- [ ] `grpcerr`: interceptors + status + RetryInfo
- [ ] `wserr`: close-code mapping + reason

### Phase 4 — logger adapters
- [ ] `adapter/slog`, `adapter/zap`, `adapter/logrus`, `adapter/zerolog`

### Phase 5 — quality layer
- [ ] `StrictMode()` validations
- [ ] cookbook docs
- [ ] (fast-follow) `cmd/herrlint`

### Phase 6 — docs
- [ ] package-level GoDoc on every package
- [ ] `README.md` with quickstart + examples
- [ ] `doc.go` examples (`ExampleXxx`)

## Decisions log (so we don't re-litigate)
- Public surface = `{Title, Message, Metadata}`. Reassurance removed. Metadata stays
  INSIDE Public (single auditable boundary).
- `Retry` tri-state (`RetryUnset/Yes/No`); absent on wire = unknown, never "false".
- herr ships no frontend/presentation; flat response body.
- Logger/i18n agnostic; core has zero third-party deps.

## Session notes
- 2026-06-11: module init + handoff doc created. Starting Phase 1 core, TDD.
