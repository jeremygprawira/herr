// Package zerologadapter lets a github.com/rs/zerolog Logger satisfy herr.Logger.
//
// herr's core is logger-agnostic: it knows only the tiny herr.Logger interface (one
// Log(ctx, Record) method) and never imports a concrete logging library. This sub-package
// is the thin bridge for zerolog — it takes a zerolog.Logger you already configured and
// returns a herr.Logger that emits one event per error, mapping each piece of the
// herr.Record to a field using the same snake_case keys the core uses elsewhere
// (code, internal, trace_id, status, cause, stack).
//
// It is its OWN Go module so importing it never pulls zerolog into a project that only
// wants the dependency-free herr core. Read top to bottom: New stores the logger, Log
// performs the emit.
package zerologadapter

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/jeremygprawira/herr"
)

// logger is the unexported adapter type. It holds the zerolog.Logger (a value type, by
// design) and exists only to attach the Log method that satisfies herr.Logger; callers
// receive it through the herr.Logger interface returned by New.
type logger struct {
	l zerolog.Logger
}

// New wraps a zerolog.Logger so it can be handed to herr's transport adapters as a
// herr.Logger sink. zerolog's Logger is a value (not a pointer), so there is no nil to
// guard; a zero-value Logger simply discards output, which is zerolog's own default.
func New(l zerolog.Logger) herr.Logger {
	return &logger{l: l}
}

// Log emits exactly one zerolog event for the given herr.Record.
//
// Flow: start an event at the level chosen from the HTTP status (server faults are louder),
// chain the always-present code plus the optional identifying/diagnostic fields (only when
// meaningful) and each internal field, then dispatch with the code as the message.
func (a *logger) Log(ctx context.Context, rec herr.Record) {
	ev := a.event(rec.HTTPStatus).Ctx(ctx).Str("code", rec.Code)
	if rec.Internal != "" {
		ev = ev.Str("internal", rec.Internal)
	}
	if rec.TraceID != "" {
		ev = ev.Str("trace_id", rec.TraceID)
	}
	if rec.HTTPStatus != 0 {
		ev = ev.Int("status", rec.HTTPStatus)
	}
	if rec.Cause != nil {
		ev = ev.Str("cause", rec.Cause.Error())
	}
	if rec.Stack != "" {
		ev = ev.Str("stack", rec.Stack)
	}
	// Each internal structured field becomes its own event field, keyed by the field's key.
	for _, f := range rec.Fields {
		ev = ev.Interface(f.Key, f.Val)
	}
	ev.Msg(rec.Code)
}

// event starts a zerolog event at the level dictated by herr's auto-log policy: server
// faults (HTTP status 500 and above) are real incidents and log at Error; everything else —
// client errors and the unset/zero status — logs at Warn.
func (a *logger) event(httpStatus int) *zerolog.Event {
	if httpStatus >= 500 {
		return a.l.Error()
	}
	return a.l.Warn()
}
