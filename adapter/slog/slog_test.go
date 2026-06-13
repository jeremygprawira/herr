// Package slogadapter_test is the black-box test suite for the slog logger adapter.
//
// It exercises ONLY the public surface of the adapter (the New constructor and the
// herr.Logger interface it returns) together with the real herr core. There are no
// mocks of herr itself; the only test double is captureHandler below, which stands in
// for the standard library's slog sink so the test can inspect exactly what the adapter
// emitted (level, message, attributes). That is fair game — it is a double of the STDLIB
// destination, not of the code under test.
package slogadapter_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/jeremygeraldprawira/herr"
	slogadapter "github.com/jeremygeraldprawira/herr/adapter/slog"
)

// captureHandler is a minimal slog.Handler test double that records every record it is
// asked to handle. The adapter emits through a *slog.Logger built on top of this handler,
// so after a Log call the test can read back the level, message, and flattened attributes
// to assert the adapter's behavior end to end.
type captureHandler struct {
	records []captured
}

// captured is one handled record, flattened to the pieces the tests assert on: the log
// level, the message string, and the attributes collapsed into a key→value map.
type captured struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// Enabled reports whether a level is active. The double accepts every level so no record
// is silently dropped before the test can see it.
func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

// Handle flattens the incoming slog.Record into a captured entry and stores it. It walks
// every attribute so the test sees the exact key/value pairs the adapter produced.
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	c := captured{level: r.Level, msg: r.Message, attrs: map[string]any{}}
	r.Attrs(func(a slog.Attr) bool {
		c.attrs[a.Key] = a.Value.Any()
		return true
	})
	h.records = append(h.records, c)
	return nil
}

// WithAttrs / WithGroup are required by slog.Handler but unused by the adapter, so the
// double simply returns itself.
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

// newCapture wires a capturing handler to a *slog.Logger and returns both, so a test can
// drive the adapter through the logger and then read what landed in the handler.
func newCapture() (*captureHandler, *slog.Logger) {
	h := &captureHandler{}
	return h, slog.New(h)
}

// TestLog_ServerFaultLogsAtErrorWithCode is the first vertical slice: a 5xx Record must be
// logged at slog.LevelError and must carry the error code as a "code" attribute.
func TestLog_ServerFaultLogsAtErrorWithCode(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{Code: "BOOM", HTTPStatus: 500})

	if len(h.records) != 1 {
		t.Fatalf("want 1 record, got %d", len(h.records))
	}
	rec := h.records[0]
	if rec.level != slog.LevelError {
		t.Errorf("want level Error, got %v", rec.level)
	}
	if rec.attrs["code"] != "BOOM" {
		t.Errorf("want code=BOOM, got %v", rec.attrs["code"])
	}
}

// TestLog_ClientErrorLogsAtWarn checks the other side of the level policy: a non-5xx
// Record (here a 404 client error) must be logged at slog.LevelWarn, not Error.
func TestLog_ClientErrorLogsAtWarn(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{Code: "NOPE", HTTPStatus: 404})

	if len(h.records) != 1 {
		t.Fatalf("want 1 record, got %d", len(h.records))
	}
	if h.records[0].level != slog.LevelWarn {
		t.Errorf("want level Warn, got %v", h.records[0].level)
	}
}

// TestLog_DetailAttrsPresentWhenSet verifies that internal, trace_id, status, and cause
// are emitted under their snake_case keys when the Record carries them — and that cause
// is rendered via its Error() string (not the error value itself).
func TestLog_DetailAttrsPresentWhenSet(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{
		Code:       "BOOM",
		HTTPStatus: 500,
		Internal:   "db connection refused",
		TraceID:    "trace-123",
		Cause:      errors.New("dial tcp: refused"),
	})

	a := h.records[0].attrs
	if a["internal"] != "db connection refused" {
		t.Errorf("want internal set, got %v", a["internal"])
	}
	if a["trace_id"] != "trace-123" {
		t.Errorf("want trace_id set, got %v", a["trace_id"])
	}
	if a["status"] != int64(500) {
		t.Errorf("want status=500, got %v (%T)", a["status"], a["status"])
	}
	if a["cause"] != "dial tcp: refused" {
		t.Errorf("want cause=error string, got %v", a["cause"])
	}
}

// TestLog_DetailAttrsOmittedWhenEmpty verifies the converse: an empty/zero Record carries
// only "code"; internal, trace_id, status, and cause are absent so log lines stay lean.
func TestLog_DetailAttrsOmittedWhenEmpty(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{Code: "BARE"})

	a := h.records[0].attrs
	for _, k := range []string{"internal", "trace_id", "status", "cause"} {
		if _, ok := a[k]; ok {
			t.Errorf("want %q omitted, but present: %v", k, a[k])
		}
	}
	if a["code"] != "BARE" {
		t.Errorf("want code=BARE, got %v", a["code"])
	}
}

// TestLog_FieldsBecomeAttrs verifies each internal structured Field is emitted as its own
// attribute keyed by Field.Key, carrying Field.Val unchanged.
func TestLog_FieldsBecomeAttrs(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{
		Code: "BOOM",
		Fields: []herr.Field{
			{Key: "user_id", Val: int64(42)},
			{Key: "region", Val: "us-east-1"},
		},
	})

	a := h.records[0].attrs
	if a["user_id"] != int64(42) {
		t.Errorf("want user_id=42, got %v (%T)", a["user_id"], a["user_id"])
	}
	if a["region"] != "us-east-1" {
		t.Errorf("want region=us-east-1, got %v", a["region"])
	}
}

// TestLog_StackPresentWhenSet verifies a captured call stack is emitted under "stack"
// when the Record carries one, and omitted otherwise.
func TestLog_StackPresentWhenSet(t *testing.T) {
	h, l := newCapture()
	log := slogadapter.New(l)

	log.Log(context.Background(), herr.Record{Code: "BOOM", Stack: "goroutine 1 ..."})
	if got := h.records[0].attrs["stack"]; got != "goroutine 1 ..." {
		t.Errorf("want stack set, got %v", got)
	}

	log.Log(context.Background(), herr.Record{Code: "BARE"})
	if _, ok := h.records[1].attrs["stack"]; ok {
		t.Errorf("want stack omitted when empty")
	}
}
