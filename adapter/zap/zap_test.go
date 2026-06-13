package zapadapter_test

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/jeremygeraldprawira/herr"
	zapadapter "github.com/jeremygeraldprawira/herr/adapter/zap"
)

// newObserved builds a herr.Logger backed by an in-memory zap core, returning both so a
// test can drive Log and then inspect the captured entries + fields.
func newObserved(t *testing.T) (herr.Logger, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	return zapadapter.New(zap.New(core)), logs
}

// fieldMap flattens an observed entry's structured fields into a plain map for assertions.
func fieldMap(e observer.LoggedEntry) map[string]any {
	m := make(map[string]any, len(e.Context))
	for _, f := range e.Context {
		m[f.Key] = f.Interface
		// zap stores strings/ints in dedicated members, not Interface.
		switch f.Type {
		case zapcore.StringType:
			m[f.Key] = f.String
		case zapcore.Int64Type, zapcore.Int32Type:
			m[f.Key] = f.Integer
		}
	}
	return m
}

// TestLog_ServerFaultIsError proves the level policy's loud end: a 5xx Record logs at Error
// and always carries the code field — so server faults surface as incidents in zap.
func TestLog_ServerFaultIsError(t *testing.T) {
	log, logs := newObserved(t)

	log.Log(context.Background(), herr.LogRecord(
		herr.New("BOOM").Kind(herr.KindInternal),
	))

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want Error", entries[0].Level)
	}
	if got := fieldMap(entries[0])["code"]; got != "BOOM" {
		t.Errorf("code field = %v, want BOOM", got)
	}
}

// TestLog_ClientErrorIsWarn proves the quiet end of the policy: a 4xx Record logs at Warn,
// so expected client errors don't masquerade as incidents.
func TestLog_ClientErrorIsWarn(t *testing.T) {
	log, logs := newObserved(t)
	log.Log(context.Background(), herr.LogRecord(herr.New("NOPE").Kind(herr.KindNotFound)))

	if got := logs.All()[0].Level; got != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", got)
	}
}

// TestLog_OptionalFields proves the present-when-set / omitted-when-empty rule for the
// optional fields, keeping lines lean.
func TestLog_OptionalFields(t *testing.T) {
	log, logs := newObserved(t)

	rich := herr.New("DB_DOWN").Kind(herr.KindUnavailable).
		Internal("connection refused").
		With("shard", "eu-3").
		Trace("trace-xyz")
	log.Log(context.Background(), herr.LogRecord(rich))

	m := fieldMap(logs.All()[0])
	if m["internal"] != "connection refused" {
		t.Errorf("internal = %v, want it present", m["internal"])
	}
	if m["trace_id"] != "trace-xyz" {
		t.Errorf("trace_id = %v, want it present", m["trace_id"])
	}
	if m["status"] != int64(503) {
		t.Errorf("status = %v, want 503", m["status"])
	}
	if m["shard"] != "eu-3" { // an internal field becomes its own zap field
		t.Errorf("shard field = %v, want eu-3", m["shard"])
	}

	// A bare error omits the optional keys entirely.
	log2, logs2 := newObserved(t)
	log2.Log(context.Background(), herr.LogRecord(herr.New("X").Kind(herr.KindNotFound)))
	bare := fieldMap(logs2.All()[0])
	if _, present := bare["internal"]; present {
		t.Error("internal should be omitted when empty")
	}
	if _, present := bare["trace_id"]; present {
		t.Error("trace_id should be omitted when empty")
	}
}

// TestLog_StackWhenPresent proves a captured stack rides into logs (and only logs).
func TestLog_StackWhenPresent(t *testing.T) {
	log, logs := newObserved(t)
	withStack := herr.New("BOOM").Kind(herr.KindInternal).WithStack()
	log.Log(context.Background(), herr.LogRecord(withStack))

	if s, _ := fieldMap(logs.All()[0])["stack"].(string); s == "" {
		t.Error("stack field should be present for a server-fault WithStack error")
	}
}

// TestNew_NilLoggerIsSafe proves a nil logger never panics (falls back to a no-op).
func TestNew_NilLoggerIsSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Log on nil-backed adapter panicked: %v", r)
		}
	}()
	zapadapter.New(nil).Log(context.Background(), herr.LogRecord(herr.New("X")))
}
