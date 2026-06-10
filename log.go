package herr

import (
	"context"
	"errors"
	"log/slog"
)

// Record is the INTERNAL, log-facing view of an error — the mirror image of the wire DTO.
// Where wireError is the safe public allow-list, Record is the full operator-facing
// detail: code, classification, status, the internal message, internal fields, the
// wrapped cause, and the trace id. Logs are trusted, so everything is included here.
type Record struct {
	Code       string
	Kind       Kind
	HTTPStatus int
	Internal   string
	Fields     []Field
	Cause      error
	TraceID    string
}

// Logger is the one-method sink herr's transport adapters use for OPT-IN auto-logging.
// It is intentionally tiny so any logging library can satisfy it with a ~10-line adapter
// (see herr/adapter/*). The core never imports a concrete logger; it only knows this
// interface, which is what keeps herr logger-agnostic.
type Logger interface {
	Log(ctx context.Context, rec Record)
}

// LogRecord extracts the structured Record from any error.
//
// If err is (or wraps) a *herr.Error, its full internal detail is returned. If it is a
// plain error, a useful fallback is produced: coded "INTERNAL" with the original error
// preserved as the Cause — so logging code never needs to special-case error types.
func LogRecord(err error) Record {
	var he *Error
	if errors.As(err, &he) {
		return Record{
			Code:       he.code,
			Kind:       he.kind,
			HTTPStatus: he.HTTPStatus(),
			Internal:   he.internal,
			Fields:     he.fields,
			Cause:      he.cause,
			TraceID:    he.traceID,
		}
	}
	// Non-herr error: minimal, safe-to-log fallback.
	return Record{
		Code:       "INTERNAL",
		Kind:       KindInternal,
		HTTPStatus: 500,
		Cause:      err,
	}
}

// LogFields flattens a Record into a map[string]any, the shape favored by field-style
// loggers (e.g. logrus.WithFields). Internal structured fields are merged in at the top
// level; identifying data uses snake_case keys. Empty values are omitted to keep lines
// lean.
func LogFields(err error) map[string]any {
	rec := LogRecord(err)
	m := make(map[string]any, len(rec.Fields)+5)
	m["code"] = rec.Code
	if rec.Internal != "" {
		m["internal"] = rec.Internal
	}
	if rec.TraceID != "" {
		m["trace_id"] = rec.TraceID
	}
	if rec.HTTPStatus != 0 {
		m["status"] = rec.HTTPStatus
	}
	if rec.Cause != nil {
		m["cause"] = rec.Cause.Error()
	}
	for _, f := range rec.Fields {
		m[f.Key] = f.Val
	}
	return m
}

// Attrs adapts an error to a slice of slog.Attr so it drops straight into structured
// logging: slog.Error("request failed", herr.Attrs(err)...). It mirrors LogFields but in
// slog's native type.
func Attrs(err error) []slog.Attr {
	rec := LogRecord(err)
	attrs := make([]slog.Attr, 0, len(rec.Fields)+5)
	attrs = append(attrs, slog.String("code", rec.Code))
	if rec.Internal != "" {
		attrs = append(attrs, slog.String("internal", rec.Internal))
	}
	if rec.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", rec.TraceID))
	}
	if rec.HTTPStatus != 0 {
		attrs = append(attrs, slog.Int("status", rec.HTTPStatus))
	}
	if rec.Cause != nil {
		attrs = append(attrs, slog.String("cause", rec.Cause.Error()))
	}
	for _, f := range rec.Fields {
		attrs = append(attrs, slog.Any(f.Key, f.Val))
	}
	return attrs
}
