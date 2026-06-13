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

MORE DONE (cycles 8–15):
- [x] `RetryAfter` + `retryAfter` seconds on wire (implies retryable) (`retry.go`)
- [x] gRPC + WS code mapping, dep-free `GRPCCode` enum + `.GRPC/.WS` + accessors (`transport_codes.go`)
- [x] `.Param/.Params` + injection-safe `{name}` substitution (H3) (`template.go`)
- [x] `.Trace`/`TraceID` → `traceId` on wire (`error.go`)
- [x] logging pull API `LogRecord/LogFields/Attrs` + `Logger` interface (`log.go`)
- [x] built-in default floor messages by Kind (`defaults.go`)
- [x] `Localizer` interface + `SetLocalizer` (atomic) + resolution chain + `Body(locale)` (`localize.go`)
- [x] H5 count bounds: fields ≤64, metadata ≤64, with truncation markers (`fields.go`)

REMAINING in Phase 1 (start here, keep TDD one-test-at-a-time):
- [x] `SetDefaults(map[Kind]string)` to override the built-in floor (atomic, like SetLocalizer)
- [x] explicit message key override (`.MessageKey(k)` / `Class.MessageKey`) — resolver asks the Localizer for `effectiveMessageKey()` (explicit, else derived)
- [x] `.WithStack()` + **conditional** stack capture (server-fault kinds only — H5); exposed in `Record.Stack`; C2 gate extended to prove the stack never leaks
- [x] H5 string-length caps (truncate long internal msg / field / metadata string values; 8 KiB, rune-safe, visible marker)
- [x] random default traceID generator helper — `NewTraceID()` (16 crypto-random bytes → 32 hex chars); used by transports when unset

### Phase 1b — Validation / field errors + multi-error interop  (designed, not built)
Build in TWO TDD steps, in this order:

1. **Native field errors** ✅ DONE
   - [x] `KindUnprocessable` → HTTP 422 (gRPC InvalidArgument, WS 1008, retry No, floor msg)
   - [x] `herr.FieldError(field, code, message) *Error` — carries a `field` path; PUBLIC parts `{field, code, message}` only
   - [x] `.FieldError(field, code, msg)` builder appends children; parent renders typed top-level **`errors[]`** (NOT metadata); each message localizes via the same chain; H5 cap 100 + `_errors_truncated` marker
   - [x] Per-field internal `.With(...)` detail stays in logs, NEVER in public `errors[]` (C2 guard test)

2. **Multi-error interop (zero new deps — structural interface checks)** ✅ DONE
   - [x] `aggregateChildren` detects aggregates structurally: `interface{ Unwrap() []error }`
     (stdlib `errors.Join`) AND `interface{ WrappedErrors() []error }` (go-multierror). NO imports.
   - [x] `fieldErrors` promotes `*herr.Error` children (via `fieldEntry`, public parts only);
     NON-herr children are never promoted (C2 guard) but stay visible to logs via the cause.
   - [x] Works for `.Wrap(errors.Join(...))` and `.Wrap(multierror)`; H5-bounded combined list.

Decision rationale (don't re-litigate): `errors[]` is **front-end-facing** (the UI renders
per-field messages), NOT a developer/log channel. We do NOT depend on go-multierror
(violates zero-dep core; superseded by stdlib `errors.Join` on our Go 1.26) — we speak its
interface structurally instead. herr is "multi-error-agnostic" like it is logger/i18n-agnostic.

### Phase 2 — i18n
- [ ] `SetSupportedLocales` + `x/text/language` matching (H4) — DEFERRED (most core-entangled;
  needs a careful dep-boundary design. Basic Accept-Language parsing already in httperr.)
- [x] `localizer/mapl` (map-based) — built via TDD by subagent; defensive deep-copy + injection-safe substitute
- [ ] embedded `en` + `id` floor-message bundles

### Phase 3 — transports ✅ DONE
- [x] `httperr`: Middleware + Write + Retry-After + Accept-Language (stdlib-only root sub-package, 6 cycles)
- [x] `grpcerr`: UnaryServerInterceptor + Status mapping + RetryInfo (submodule: grpc/genproto/protobuf, 4 cycles)
- [x] `wserr`: close-code mapping + reason + ControlPayload (stdlib-only root sub-package, 2 cycles + guards)

### Phase 4 — logger adapters ✅ DONE
- [x] `adapter/slog` (stdlib `log/slog`, root sub-package, by subagent)
- [x] `adapter/zap`, `adapter/logrus`, `adapter/zerolog`  (each a submodule with replace; structural twins of slog)

#### Module layout (multi-module repo, root stays dep-free)
- ROOT module `herr` contains: core + `httperr/` + `wserr/` + `adapter/slog/` + `localizer/mapl/`
  (all stdlib-only, no separate go.mod).
- SUBMODULES (own go.mod + `replace github.com/jeremygeraldprawira/herr => ../..` or `../`):
  `adapter/zap`, `adapter/logrus`, `adapter/zerolog` (two levels → `../../`), `grpcerr` (one level → `../`).
- `go.work` at the root ties them together for local testing. Sweep:
  `for m in . adapter/zap adapter/logrus adapter/zerolog grpcerr; do (cd $m && go test ./...); done`

### Phase 5 — quality layer
- [x] `StrictMode(on bool)` — unfilled `{param}` placeholders rendered visible in strict mode (H3); atomic, never panics
- [ ] cookbook docs (README covers the common flows; a dedicated cookbook is a nice-to-have)
- [ ] (fast-follow) `cmd/herrlint` — separate tool, deferred

### Phase 6 — docs
- [x] package-level GoDoc on every package (core + httperr + wserr + 4 adapters + mapl + grpcerr all have package doc comments)
- [x] `README.md` with quickstart + examples
- [x] `doc.go` examples — `ExampleError` + `ExampleError_fieldErrors` in `example_test.go`, verified by `go test` against their `// Output`

## Decisions log (so we don't re-litigate)
- Public surface = `{Title, Message, Metadata}`. Reassurance removed. Metadata stays
  INSIDE Public (single auditable boundary).
- `Retry` tri-state (`RetryUnset/Yes/No`); absent on wire = unknown, never "false".
- herr ships no frontend/presentation; flat response body.
- Logger/i18n agnostic; core has zero third-party deps.

## Current state (read this to resume)
- **62 tests + 1 fuzz, all green.** `go test ./...`, `go test -race ./...`, `go vet ./...` all pass.
- **Phase 1 core + Phase 1b COMPLETE.** Core is dependency-free. Added since last summary:
  `WithStack` (conditional, server-fault only), H5 string caps, `NewTraceID`,
  `KindUnprocessable`, native field errors (`FieldError` + `errors[]`), multi-error interop.
- New source files: `stack.go` `trace.go` `field_errors.go` `multierror.go`.
- **Leaf packages delivered (all green under -race + vet):** `httperr/` (HTTP transport),
  `wserr/` (WebSocket transport), `adapter/slog/` (slog logger adapter), `localizer/mapl/`
  (map localizer). slog + mapl were built by parallel subagents in isolated worktrees under
  strict TDD, then reviewed + merged. Transport accessors added to core:
  `RetryAfterSeconds()`, `Retryable()`.
- **103 tests across 5 modules, all green + vet-clean:** root `herr` (85: core + httperr +
  wserr + adapter/slog + localizer/mapl), `adapter/zap` (5), `adapter/logrus` (5),
  `adapter/zerolog` (4), `grpcerr` (4).
- **Phases 1, 1b, 2 (mapl), 3, 4 DONE.** Remaining: Phase 2 leftovers (`en`/`id` floor
  bundles; `SetSupportedLocales`/x/text H4 — deferred), Phase 5 (StrictMode, cookbook,
  herrlint), Phase 6 (README, package GoDoc, `ExampleXxx`). **Resume next at Phase 5/6.**
- **Both security gates proven & continuously re-verified:** C1 (`TestCatalog_*`, race-clean)
  and C2 (`TestSafeSplit_InternalNeverLeaks` + `FuzzWire_NeverLeaksInternal`, ~8M execs/run).
- Source files (all commented): `error.go` `kind.go` `public.go` `wire.go` `fields.go`
  `catalog.go` `retry.go` `match.go` `transport_codes.go` `template.go` `log.go`
  `defaults.go` `localize.go`.
- Public API so far: `New/Define/Class/Public/Msg`, builders `Kind/Status/GRPC/WS/Title/
  Message/Public/Meta/WithPublic/With/Internal/Internalf/Param/Params/Retry/RetryAfter/
  Trace/Wrap`, accessors `Code/HTTPStatus/GRPCCode/WSClose/TraceID/Error/Unwrap/Is/Body`,
  builder `MessageKey`, funcs `LogRecord/LogFields/Attrs/SetLocalizer/SetDefaults`,
  ifaces `Localizer/Logger`. `Class` now has a `MessageKey` field too.
- **Next concrete step:** next unchecked box in Phase 1 (`.WithStack()` + conditional stack
  capture — server/Internal kinds only per H5 — exposed in `Record.Stack`), then the rest of
  Phase 1, then Phase 2 (i18n adapters incl. `id` bundle + `SetSupportedLocales` H4), then
  Phase 3 transports (httperr first — highest external value). Write ONE failing test, make
  it pass, commit.

## Session notes
- 2026-06-11: module init + handoff doc created.
- 2026-06-11: Phase 1 core — 15 TDD cycles, one commit each. 44 tests + leak fuzz green;
  -race clean; vet clean. Security gates built FIRST and re-verified after each change to
  `wire()`. Core remains dependency-free (`go.mod` has no third-party requires).
- 2026-06-13: `SetDefaults(map[Kind]string)` added via TDD (atomic, like `SetLocalizer`;
  per-Kind override of the floor, nil restores built-in, explicit message still wins). While
  re-verifying C2, the leak fuzz surfaced a harness false positive: a NUL byte in the PUBLIC
  `code` escapes to ` `, whose text contains a secret like "0000". Fixed the fuzz to
  discount matches explained by encoded public content (no real leak; C2 invariant unchanged)
  and kept the seed as a regression corpus entry. 46 tests + fuzz green; -race & vet clean.
- 2026-06-13: explicit message-key override added via TDD (2 cycles: `.MessageKey(k)` builder,
  then `Class.MessageKey` field copied by `New()`). `resolveMessage` now asks the Localizer
  for `effectiveMessageKey()` (explicit beats derived, one place owns the rule). Inline
  message still wins; catalog message is still the literal fallback. 48 tests + fuzz green;
  -race & vet clean; C2 leak fuzz re-verified after touching `catalog.go`.
