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
	attrs := []slog.Attr{slog.String("code", rec.Code)}
	a.l.LogAttrs(ctx, slog.LevelError, rec.Code, attrs...)
}
