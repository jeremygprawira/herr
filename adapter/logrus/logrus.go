// Package logrusadapter lets a github.com/sirupsen/logrus *logrus.Logger satisfy
// herr.Logger.
//
// herr's core is logger-agnostic: it knows only the tiny herr.Logger interface (one
// Log(ctx, Record) method) and never imports a concrete logging library. This sub-package
// is the thin bridge for logrus — it takes a *logrus.Logger you already configured and
// returns a herr.Logger that emits one entry per error, mapping each piece of the
// herr.Record to a logrus field using the same snake_case keys the core uses elsewhere
// (code, internal, trace_id, status, cause, stack).
//
// It is its OWN Go module so importing it never pulls logrus into a project that only wants
// the dependency-free herr core. Read top to bottom: New wraps the logger, Log performs the
// emit.
package logrusadapter

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/jeremygeraldprawira/herr"
)

// logger is the unexported adapter type. It holds the wrapped *logrus.Logger and exists only
// to attach the Log method that satisfies herr.Logger; callers receive it through the
// herr.Logger interface returned by New and never see this concrete type.
type logger struct {
	l *logrus.Logger
}

// New wraps a *logrus.Logger so it can be handed to herr's transport adapters as a
// herr.Logger sink. Every error herr decides to auto-log then flows into Log below.
//
// A nil logger is tolerated: rather than panic on the request path of a misconfigured
// caller, New substitutes a fresh logrus.New() so errors still go somewhere.
func New(l *logrus.Logger) herr.Logger {
	if l == nil {
		l = logrus.New()
	}
	return &logger{l: l}
}

// Log emits exactly one logrus entry for the given herr.Record.
//
// Flow: collect the Record's pieces into a logrus.Fields map (always code; the rest only
// when meaningful, plus each internal field), bind it (with the context) to an entry, then
// emit at Error for server faults and Warn otherwise.
func (a *logger) Log(ctx context.Context, rec herr.Record) {
	fields := logrus.Fields{"code": rec.Code}
	if rec.Internal != "" {
		fields["internal"] = rec.Internal
	}
	if rec.TraceID != "" {
		fields["trace_id"] = rec.TraceID
	}
	if rec.HTTPStatus != 0 {
		fields["status"] = rec.HTTPStatus
	}
	if rec.Cause != nil {
		fields["cause"] = rec.Cause.Error()
	}
	if rec.Stack != "" {
		fields["stack"] = rec.Stack
	}
	// Each internal structured field becomes its own logrus field, keyed by the field's
	// key, so debugging context is queryable in logs.
	for _, f := range rec.Fields {
		fields[f.Key] = f.Val
	}

	entry := a.l.WithContext(ctx).WithFields(fields)

	// Level policy: server faults (HTTP status 500 and above) are real incidents and log at
	// Error; everything else — client errors and the unset/zero status — logs at Warn.
	if rec.HTTPStatus >= 500 {
		entry.Error(rec.Code)
		return
	}
	entry.Warn(rec.Code)
}
