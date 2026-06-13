// Package zapadapter lets a go.uber.org/zap *zap.Logger satisfy herr.Logger.
//
// herr's core is logger-agnostic: it knows only the tiny herr.Logger interface (one
// Log(ctx, Record) method) and never imports a concrete logging library. This sub-package
// is the thin bridge for zap — it takes a *zap.Logger you already configured and returns a
// herr.Logger that emits one structured entry per error, mapping each piece of the
// herr.Record to a zap field using the same snake_case keys the core uses elsewhere
// (code, internal, trace_id, status, cause, stack).
//
// It is its OWN Go module so importing it never pulls zap into a project that only wants the
// dependency-free herr core. Read top to bottom: New wraps the logger, Log performs the emit.
package zapadapter

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/jeremygprawira/herr"
)

// logger is the unexported adapter type. It holds the wrapped *zap.Logger and exists only to
// attach the Log method that satisfies herr.Logger; callers receive it through the
// herr.Logger interface returned by New and never see this concrete type.
type logger struct {
	l *zap.Logger
}

// New wraps a *zap.Logger so it can be handed to herr's transport adapters as a herr.Logger
// sink. Every error herr decides to auto-log then flows into Log below.
//
// A nil logger is tolerated: rather than panic on the request path of a misconfigured
// caller, New substitutes zap.NewNop() so the program keeps running (silently).
func New(l *zap.Logger) herr.Logger {
	if l == nil {
		l = zap.NewNop()
	}
	return &logger{l: l}
}

// Log emits exactly one structured zap entry for the given herr.Record.
//
// Flow: choose the level from the HTTP status (server faults are louder), build the field
// list starting with the always-present code, add the optional identifying/diagnostic
// fields only when meaningful, append each internal field, then emit at the chosen level.
func (a *logger) Log(_ context.Context, rec herr.Record) {
	level := levelFor(rec.HTTPStatus)

	// code is the one field always present; the rest are added only when set so log lines
	// stay lean and operators are not fed empty keys.
	fields := []zap.Field{zap.String("code", rec.Code)}
	if rec.Internal != "" {
		fields = append(fields, zap.String("internal", rec.Internal))
	}
	if rec.TraceID != "" {
		fields = append(fields, zap.String("trace_id", rec.TraceID))
	}
	if rec.HTTPStatus != 0 {
		fields = append(fields, zap.Int("status", rec.HTTPStatus))
	}
	if rec.Cause != nil {
		fields = append(fields, zap.String("cause", rec.Cause.Error()))
	}
	if rec.Stack != "" {
		fields = append(fields, zap.String("stack", rec.Stack))
	}

	// Each internal structured field becomes its own zap field, keyed by the field's key,
	// so debugging context (user ids, regions, ...) is queryable in logs.
	for _, f := range rec.Fields {
		fields = append(fields, zap.Any(f.Key, f.Val))
	}

	a.l.Log(level, rec.Code, fields...)
}

// levelFor applies herr's auto-log level policy: server faults (HTTP status 500 and above)
// are real incidents and log at Error; everything else — client errors and the unset/zero
// status — logs at Warn.
func levelFor(httpStatus int) zapcore.Level {
	if httpStatus >= 500 {
		return zapcore.ErrorLevel
	}
	return zapcore.WarnLevel
}
