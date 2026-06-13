package herr_test

import (
	"errors"
	"testing"

	"github.com/jeremygprawira/herr"
)

// TestLogRecord_CarriesInternalDetail proves the "developer-friendly" half: LogRecord
// surfaces EVERYTHING an operator needs — code, status, the internal message, internal
// fields, the wrapped cause, and the trace id. Unlike the wire body, logs are trusted, so
// internal detail is intentionally present here.
func TestLogRecord_CarriesInternalDetail(t *testing.T) {
	cause := errors.New("pq: deadlock detected")
	e := herr.New("ACCOUNT_CONNECT_FAILED").
		Kind(herr.KindUnavailable).
		Internal("kyc upstream failed").
		With("upstream", "kyc-svc").
		Trace("t-1").
		Wrap(cause)

	rec := herr.LogRecord(e)

	if rec.Code != "ACCOUNT_CONNECT_FAILED" {
		t.Errorf("Code = %q", rec.Code)
	}
	if rec.Internal != "kyc upstream failed" {
		t.Errorf("Internal = %q", rec.Internal)
	}
	if rec.HTTPStatus != 503 {
		t.Errorf("HTTPStatus = %d, want 503", rec.HTTPStatus)
	}
	if rec.TraceID != "t-1" {
		t.Errorf("TraceID = %q", rec.TraceID)
	}
	if !errors.Is(rec.Cause, cause) {
		t.Errorf("Cause = %v, want %v", rec.Cause, cause)
	}
	var foundUpstream bool
	for _, f := range rec.Fields {
		if f.Key == "upstream" && f.Val == "kyc-svc" {
			foundUpstream = true
		}
	}
	if !foundUpstream {
		t.Errorf("internal field 'upstream' missing from %v", rec.Fields)
	}
}

// TestLogFields_FlatMap proves the map shape (for logrus-style loggers) contains the
// identifying keys plus the internal fields flattened in.
func TestLogFields_FlatMap(t *testing.T) {
	e := herr.New("X").Internal("boom").With("k", "v").Trace("t")
	m := herr.LogFields(e)

	if m["code"] != "X" {
		t.Errorf("code = %v", m["code"])
	}
	if m["internal"] != "boom" {
		t.Errorf("internal = %v", m["internal"])
	}
	if m["trace_id"] != "t" {
		t.Errorf("trace_id = %v", m["trace_id"])
	}
	if m["k"] != "v" {
		t.Errorf("field k = %v", m["k"])
	}
}

// TestLogRecord_NonHerrError proves graceful degradation: a plain error still yields a
// useful record (coded INTERNAL, original message preserved as the cause) so logging code
// never has to special-case error types.
func TestLogRecord_NonHerrError(t *testing.T) {
	rec := herr.LogRecord(errors.New("something failed"))
	if rec.Code != "INTERNAL" {
		t.Errorf("Code = %q, want INTERNAL", rec.Code)
	}
	if rec.Cause == nil || rec.Cause.Error() != "something failed" {
		t.Errorf("Cause = %v, want the original error", rec.Cause)
	}
}

// TestAttrs_IncludesCode proves the slog bridge produces attributes (at minimum the
// code) so `slog.Error("msg", herr.Attrs(err)...)` works.
func TestAttrs_IncludesCode(t *testing.T) {
	attrs := herr.Attrs(herr.New("X").Internal("boom"))
	var hasCode bool
	for _, a := range attrs {
		if a.Key == "code" && a.Value.String() == "X" {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("Attrs missing code attr: %v", attrs)
	}
}
