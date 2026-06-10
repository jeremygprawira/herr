# CLAUDE.md — herr

Guidance for Claude Code working in this repository. **Read `CONTINUE.md` first** — it is
the live progress tracker and tells you exactly where to resume.

## What this is
`herr` is a framework-, logger-, and i18n-agnostic Go error library. One error carries a
**public** surface (safe, localized, user-facing — crosses the wire) and an **internal**
surface (rich detail — logs only), with the boundary enforced structurally. Full design:
`docs/specs/2026-06-11-herr-design.md` (final, no open questions).

## Golden rules (do not drift)
1. **Strict TDD, vertical slices.** One failing test → minimal code to pass → refactor →
   commit. Never write a batch of tests ahead of implementation. Black-box tests live in
   package `herr_test` and exercise only the public API.
2. **Comment every file** with package/type/func doc comments that explain the *flow* of
   the process — a junior should follow it top to bottom.
3. **Security gates are sacred.** Two invariants must never regress:
   - **C2 (no leaks):** only the `wire()` allow-list crosses the wire; internal data lives
     in unexported fields; `MarshalJSON` delegates to `wire()`. Guarded by
     `TestSafeSplit_InternalNeverLeaks` + `FuzzWire_NeverLeaksInternal`.
   - **C1 (no shared mutable state):** `Class` is immutable; per-request state lives only
     on the `*Error` instance. Guarded by `TestCatalog_*` under `-race`.
   If you touch serialization or the catalog, run the leak fuzz and `-race`.
4. **Convention with override.** Defaults derive from `Kind`/`Code`; every default is
   overridable; magic is always a fallback, never a trap.
5. **Commit per green cycle** with a clear message. End commit messages with the
   Co-Authored-By trailer.
6. **Core has zero third-party deps.** Logging/i18n/transport integrations go in
   sub-packages behind tiny interfaces. Keep `go.mod` clean for the root package.

## Commands
```bash
go test ./...            # all tests
go test -race ./...      # C1 gate
go test -run=xxx -fuzz=FuzzWire_NeverLeaksInternal -fuzztime=10s .   # C2 gate
go vet ./...
```

## Build order
See `CONTINUE.md` → "Build order". Always resume from the first unchecked box.

## Key design decisions (settled — don't re-litigate)
- Public surface = `{Title, Message, Metadata}` (Reassurance removed; Metadata stays
  INSIDE `Public` for one auditable boundary).
- `Retry` is tri-state (`RetryUnset/Yes/No`); absent on the wire = unknown, never "false".
- herr ships **no** frontend/presentation; the response body is flat data.
- Public messages localize; internal/log text stays one language.
