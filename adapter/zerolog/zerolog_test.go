package zerologadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"

	"github.com/jeremygprawira/herr"
	zerologadapter "github.com/jeremygprawira/herr/adapter/zerolog"
)

// newCaptured builds a herr.Logger backed by a zerolog logger writing JSON to a buffer, so a
// test can drive Log and decode the single emitted line.
func newCaptured(t *testing.T) (herr.Logger, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	return zerologadapter.New(zerolog.New(buf)), buf
}

func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("log line is not valid JSON: %v (raw: %s)", err, buf.String())
	}
	return m
}

// TestLog_ServerFaultIsError proves the loud end of the level policy: a 5xx Record logs at
// "error" level and carries the code field.
func TestLog_ServerFaultIsError(t *testing.T) {
	log, buf := newCaptured(t)
	log.Log(context.Background(), herr.LogRecord(herr.New("BOOM").Kind(herr.KindInternal)))

	m := decodeLine(t, buf)
	if m["level"] != "error" {
		t.Errorf("level = %v, want error", m["level"])
	}
	if m["code"] != "BOOM" {
		t.Errorf("code = %v, want BOOM", m["code"])
	}
}

// TestLog_ClientErrorIsWarn proves the quiet end: a 4xx Record logs at "warn" level.
func TestLog_ClientErrorIsWarn(t *testing.T) {
	log, buf := newCaptured(t)
	log.Log(context.Background(), herr.LogRecord(herr.New("NOPE").Kind(herr.KindNotFound)))

	if m := decodeLine(t, buf); m["level"] != "warn" {
		t.Errorf("level = %v, want warn", m["level"])
	}
}

// TestLog_OptionalFields proves present-when-set / omitted-when-empty and that each internal
// field becomes its own event field.
func TestLog_OptionalFields(t *testing.T) {
	log, buf := newCaptured(t)
	rich := herr.New("DB_DOWN").Kind(herr.KindUnavailable).
		Internal("connection refused").
		With("shard", "eu-3").
		Trace("trace-xyz")
	log.Log(context.Background(), herr.LogRecord(rich))

	m := decodeLine(t, buf)
	if m["internal"] != "connection refused" {
		t.Errorf("internal = %v, want present", m["internal"])
	}
	if m["trace_id"] != "trace-xyz" {
		t.Errorf("trace_id = %v, want present", m["trace_id"])
	}
	if m["status"] != float64(503) {
		t.Errorf("status = %v, want 503", m["status"])
	}
	if m["shard"] != "eu-3" {
		t.Errorf("shard = %v, want eu-3", m["shard"])
	}

	log2, buf2 := newCaptured(t)
	log2.Log(context.Background(), herr.LogRecord(herr.New("X").Kind(herr.KindNotFound)))
	bare := decodeLine(t, buf2)
	if _, present := bare["internal"]; present {
		t.Error("internal should be omitted when empty")
	}
	if _, present := bare["trace_id"]; present {
		t.Error("trace_id should be omitted when empty")
	}
}

// TestLog_StackWhenPresent proves a captured stack rides into logs.
func TestLog_StackWhenPresent(t *testing.T) {
	log, buf := newCaptured(t)
	log.Log(context.Background(), herr.LogRecord(
		herr.New("BOOM").Kind(herr.KindInternal).WithStack(),
	))
	if s, _ := decodeLine(t, buf)["stack"].(string); s == "" {
		t.Error("stack should be present for a server-fault WithStack error")
	}
}
