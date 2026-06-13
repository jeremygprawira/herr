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
