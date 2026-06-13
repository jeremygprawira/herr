// Package slogadapter lets a standard library *log/slog.Logger satisfy herr.Logger.
//
// herr's core is logger-agnostic: it knows only the tiny herr.Logger interface
// (one Log(ctx, Record) method) and never imports a concrete logging library. This
// sub-package is the thin bridge for slog — it takes a *slog.Logger you already
// configured and returns a herr.Logger that emits one structured record per error,
// with each piece of the herr.Record mapped to a slog attribute using the same
// snake_case keys the core uses elsewhere (code, internal, trace_id, status, cause).
//
// Read top to bottom: New wraps the logger, and Log performs the single emit.
package slogadapter

import (
	"context"
	"log/slog"

	"github.com/jeremygeraldprawira/herr"
)

// logger is the unexported adapter type. It holds the wrapped *slog.Logger and exists
// only to attach the Log method that satisfies herr.Logger; callers receive it through
// the herr.Logger interface returned by New and never see this concrete type.
type logger struct {
	l *slog.Logger
}

// New wraps a *slog.Logger so it can be handed to herr's transport adapters as a
// herr.Logger sink. Every error herr decides to auto-log then flows into Log below.
func New(l *slog.Logger) herr.Logger {
	return &logger{l: l}
}

// Log emits exactly one structured slog record for the given herr.Record.
//
// Flow: choose the level from the HTTP status (server faults are louder), build the
// attribute list starting with the always-present code, then emit through the wrapped
// logger using the supplied context.
func (a *logger) Log(ctx context.Context, rec herr.Record) {
	level := levelFor(rec.HTTPStatus)

	// code is the one attribute always present; the rest are added only when meaningful
	// so log lines stay lean and operators are not fed empty keys.
	attrs := []slog.Attr{slog.String("code", rec.Code)}
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
	if rec.Stack != "" {
		attrs = append(attrs, slog.String("stack", rec.Stack))
	}

	// Each internal structured field becomes its own top-level attribute, keyed by the
	// field's own key, so debugging context (user ids, regions, ...) is queryable in logs.
	for _, f := range rec.Fields {
		attrs = append(attrs, slog.Any(f.Key, f.Val))
	}

	a.l.LogAttrs(ctx, level, rec.Code, attrs...)
}

// levelFor applies herr's auto-log level policy: server faults (HTTP status 500 and
// above) are real incidents and log at Error; everything else — client errors and the
// unset/zero status — logs at Warn.
func levelFor(httpStatus int) slog.Level {
	if httpStatus >= 500 {
		return slog.LevelError
	}
	return slog.LevelWarn
}
