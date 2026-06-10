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

### Phase 1 — core (`herr` package)  ← CURRENT (~60% done)
DONE (7 TDD cycles, 25 tests + 1 fuzz, all green, `-race` clean, `go vet` clean):
- [x] `Kind` enum + default **HTTP** mapping + Kind→retryability (`kind.go`, `retry.go`)
- [x] `Error` type: `New(code)`, `Error()`, `Code()`, `Unwrap()` (`error.go`)
- [x] `Public` struct {Title, Message, Metadata} + `Msg()` shorthand (`public.go`)
- [x] builder so far: `.Kind/.Status/.Title/.Message/.Public/.Meta/.WithPublic/.With/.Internal/.Internalf/.Wrap/.Retry`
- [x] **C2 leak gate:** `wire()` DTO + `MarshalJSON` allowlist + leak fuzz (`wire.go`, `leak_test.go`) — ~490k execs, 0 leaks
- [x] wrapping + `errors.Is`/`As` by code + catalog `Class.Is` (`match.go`)
- [x] catalog: `Define(Class)` + `.New()` (`catalog.go`)
- [x] **C1 race gate:** concurrent New()/render under `-race` (`catalog_test.go`)
- [x] `Retry` tri-state → wire OMITS when unknown (`retry.go`)

REMAINING in Phase 1 (start here, keep TDD one-test-at-a-time):
- [ ] `RetryAfter time.Duration` field + builder `.RetryAfter()` + wire field `retryAfter`
- [ ] gRPC + WS code mapping (`kindGRPC`, `kindWS`) + `.GRPC()/.WS()` overrides + `GRPCCode()/WSClose()` accessors
- [ ] message templating: `.Param/.Params` + `{name}` substitution (named-placeholder ONLY, never Sprintf-with-user-format — H3) applied to Title/Message in `wire()`
- [ ] `.Trace(id)` + traceID in wire (`traceId`); random default (riders)
- [ ] `.WithStack()` + **conditional** stack capture (server/Internal kinds only — H5)
- [ ] message resolution chain §7 + `Localizer` interface + `SetLocalizer`
- [ ] built-in default messages (en) keyed by Kind + `SetDefaults` (floor)
- [ ] H5 bounds: cap fields ≤64, metadata ≤64, string lengths, with truncation markers
- [ ] logging pull API: `Attrs(err)/LogFields(err)/LogRecord(err)` + one-method `Logger` interface
- [ ] global config via `atomic.Value`, init-time only (riders)

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

## Current state (read this to resume)
- **25 tests + 1 fuzz, all green.** `go test ./...`, `go test -race ./...`, `go vet ./...` all pass.
- **Both security gates proven in code:** C1 (`TestCatalog_*`, race-clean) and C2
  (`TestSafeSplit_InternalNeverLeaks` + `FuzzWire_NeverLeaksInternal`).
- Files: `error.go` `kind.go` `public.go` `wire.go` `fields.go` `catalog.go` `retry.go`
  `match.go` (+ matching `*_test.go`). Every file has package/type/func comments.
- **Next concrete step:** `RetryAfter` (smallest slice) or gRPC/WS mapping. Pick the top
  unchecked box in Phase 1, write ONE failing test, make it pass, commit.

## Session notes
- 2026-06-11: module init + handoff doc created.
- 2026-06-11: Phase 1 core — 7 TDD cycles. Error/Kind/Public/wire(C2)/catalog(C1)/Retry/
  Is-As. 25 tests + leak fuzz green; -race clean. Security gates done first, as planned.
