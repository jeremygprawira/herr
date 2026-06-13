# herr - humanized error

A framework-, logger-, and i18n-agnostic Go error library. One error value carries **two
surfaces at once**:

- a **public** surface — safe, localized, user-facing data that may cross the wire, and
- an **internal** surface — rich detail (cause, fields, stack) that goes **only** to logs.

The boundary between them is **structural**: internal data lives in unexported fields and is
never serialized, so you cannot leak it by accident. The core has **zero third-party
dependencies**; every logger, transport, and i18n integration is an optional sub-package
behind a tiny interface.

```go
err := herr.New("ORDER_NOT_FOUND").
    Kind(herr.KindNotFound).
    Public(herr.Msg("We couldn't find that order.")).
    Internal("order 4821 missing from shard eu-3").  // logs only
    With("shard", "eu-3").                            // logs only
    Wrap(cause)                                       // logs only

httperr.Write(w, r, err) // → 404 + {"code":"ORDER_NOT_FOUND","message":"We couldn't find that order."}
```

The client sees the safe body. The internal message, fields, and cause never leave your
logs.

## Why

Most error libraries make you choose between *developer detail* and *user safety*, then trust
every call site to keep them apart. herr makes the split a type-system property:

- **You can't leak internals.** The response body is built by hand from an allow-list
  (`wireError`), never by reflecting over the error. A fuzz test injects secrets into every
  internal channel and asserts they never appear in any render (gate **C2**).
- **No shared mutable state.** Error *classes* are immutable; all per-request state lives on
  the instance. Verified under `-race` (gate **C1**).
- **Convention with override.** Classify once by `Kind`; the right HTTP status, gRPC code,
  WebSocket close code, and retryability fall out by default — each overridable.

## Install

```bash
go get github.com/jeremygprawira/herr
```

The core is dependency-free. Optional integrations are separate modules, e.g.:

```bash
go get github.com/jeremygprawira/herr/adapter/zap
go get github.com/jeremygprawira/herr/grpcerr
```

## Core concepts

### Kinds drive the defaults

| Kind | HTTP | gRPC | Retry |
|------|------|------|-------|
| `KindInternal` (zero value) | 500 | Internal | unknown |
| `KindInvalid` | 400 | InvalidArgument | no |
| `KindUnprocessable` | 422 | InvalidArgument | no |
| `KindUnauthorized` | 401 | Unauthenticated | no |
| `KindForbidden` | 403 | PermissionDenied | no |
| `KindNotFound` | 404 | NotFound | no |
| `KindConflict` | 409 | Aborted | no |
| `KindRateLimited` | 429 | ResourceExhausted | yes |
| `KindTimeout` | 504 | DeadlineExceeded | yes |
| `KindUnavailable` | 503 | Unavailable | yes |

The zero value is `KindInternal` on purpose: an unclassified error is treated as a server
fault (500), never a silent success.

### Inline vs. catalog

Author errors inline:

```go
herr.New("RATE_LIMITED").Kind(herr.KindRateLimited).RetryAfter(30 * time.Second)
```

…or define a reusable, immutable class once and stamp instances from it:

```go
var ErrNotFound = herr.Define(herr.Class{
    Code:   "NOT_FOUND",
    Kind:   herr.KindNotFound,
    Public: herr.Public{Message: "Resource not found."},
})

func get(id string) error {
    return ErrNotFound.New().With("id", id) // fresh instance, own per-request state
}

if ErrNotFound.Is(err) { /* ... */ }        // matches through wrapping
```

### Public vs. internal

```go
e := herr.New("PAYMENT_FAILED").Kind(herr.KindUnavailable).
    Public(herr.Msg("We couldn't process your payment. Please try again.")).
    WithPublic("support_url", "https://help.example.com/payments"). // public metadata
    Internal("stripe: card_declined insufficient_funds").           // logs only
    With("stripe_request_id", "req_123").                           // logs only
    Wrap(stripeErr)                                                 // logs only
```

`Title`, `Message`, and `Metadata` are the only things that cross the wire. Everything else
serves logs.

### Field errors (validation)

```go
err := herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).
    FieldError("email", "INVALID_EMAIL", "Enter a valid email address.").
    FieldError("age", "OUT_OF_RANGE", "You must be 18 or older.")
```

renders a typed, top-level `errors[]` your front end can map to inputs:

```json
{
  "code": "VALIDATION_FAILED",
  "errors": [
    {"field": "email", "code": "INVALID_EMAIL", "message": "Enter a valid email address."},
    {"field": "age",   "code": "OUT_OF_RANGE",  "message": "You must be 18 or older."}
  ]
}
```

herr is also **multi-error-agnostic**: wrap a `errors.Join(...)` (or a `go-multierror`) and
its `*herr.Error` children are promoted into `errors[]` automatically; non-herr children stay
in the logs and never leak.

```go
return herr.New("VALIDATION_FAILED").Kind(herr.KindUnprocessable).Wrap(errors.Join(fieldErrs...))
```

## Transports

```go
// HTTP
httperr.Write(w, r, err)              // status + safe JSON body + Retry-After + Accept-Language
mux.Handle("/", httperr.Middleware(h)) // recovers panics into a safe 500

// gRPC
return nil, grpcerr.Status(err, locale).Err()         // safe status + RetryInfo
grpc.NewServer(grpc.UnaryInterceptor(grpcerr.UnaryServerInterceptor("")))

// WebSocket
code, reason := wserr.Close(err, locale)  // RFC 6455 close code + safe reason (<=123 bytes)
payload := wserr.ControlPayload(err, locale) // ready-to-send close-frame bytes
```

## Logging

The core stays logger-agnostic behind a one-method `Logger` interface. Pull a structured
record from any error:

```go
slog.Default().LogAttrs(ctx, slog.LevelError, "request failed", herr.Attrs(err)...)
rec := herr.LogRecord(err) // {Code, Kind, HTTPStatus, Internal, Fields, Cause, TraceID, Stack}
```

Or use a ~10-line adapter so a transport can auto-log:

```go
import slogadapter "github.com/jeremygprawira/herr/adapter/slog"

logger := slogadapter.New(slog.Default()) // also: adapter/zap, adapter/logrus, adapter/zerolog
```

Adapters apply a sane policy: server faults (≥500) log at Error, everything else at Warn.

## i18n

Public messages localize; internal text stays one language. herr asks an installed
`Localizer` for `errors.<code>.message` (override per-error with `.MessageKey`), falling back
to the literal message, then a built-in per-Kind floor (overridable with `SetDefaults`).

```go
import "github.com/jeremygprawira/herr/localizer/mapl"

herr.SetLocalizer(mapl.New(map[string]map[string]string{
    "id": {"errors.not_found.message": "Tidak ditemukan."},
}))

body := err.Body("id") // transports pass the request locale (e.g. from Accept-Language)
```

## Security gates

Two invariants are protected by tests that run on every change:

- **C2 (no leaks):** only the `wire()` allow-list crosses the wire; `MarshalJSON` delegates
  to it. Guarded by `TestSafeSplit_InternalNeverLeaks` + `FuzzWire_NeverLeaksInternal`.
- **C1 (no shared mutable state):** classes are immutable; per-request state is per-instance.
  Guarded by the catalog tests under `-race`.

```bash
go test ./...                                                   # all tests
go test -race ./...                                             # C1
go test -run=xxx -fuzz=FuzzWire_NeverLeaksInternal -fuzztime=10s . # C2
```

## Module layout

```
herr/                 core (zero deps) + httperr + wserr + adapter/slog + localizer/mapl
herr/adapter/zap      separate module (go.uber.org/zap)
herr/adapter/logrus   separate module (github.com/sirupsen/logrus)
herr/adapter/zerolog  separate module (github.com/rs/zerolog)
herr/grpcerr          separate module (google.golang.org/grpc)
```

Third-party integrations are separate modules so importing the core never pulls a logging or
transport dependency into your build.

## License

[MIT](LICENSE) © Jeremy
