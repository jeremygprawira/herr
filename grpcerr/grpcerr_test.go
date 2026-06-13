package grpcerr_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jeremygeraldprawira/herr"
	"github.com/jeremygeraldprawira/herr/grpcerr"
)

// TestStatus_CodeAndMessage proves the core mapping: a herr error becomes a *status.Status
// whose gRPC code follows the error's Kind and whose message is the SAFE public message —
// the shape a gRPC client receives.
func TestStatus_CodeAndMessage(t *testing.T) {
	e := herr.New("ORDER_NOT_FOUND").Kind(herr.KindNotFound).
		Public(herr.Msg("We couldn't find that order."))

	st := grpcerr.Status(e, "")

	if st.Code() != codes.NotFound {
		t.Errorf("code = %v, want NotFound", st.Code())
	}
	if st.Message() != "We couldn't find that order." {
		t.Errorf("message = %q, want the public message", st.Message())
	}
	if strings.Contains(st.Message(), "ORDER_NOT_FOUND") {
		t.Errorf("message unexpectedly contains the code: %q", st.Message())
	}
	// status.FromError round-trips it cleanly.
	if _, ok := status.FromError(st.Err()); !ok {
		t.Error("st.Err() should be a well-formed gRPC status error")
	}
}

// TestStatus_NonHerrIsSafeInternal proves the boundary holds for a plain error: it maps to
// codes.Internal and its raw string never appears in the status message or details (C2).
func TestStatus_NonHerrIsSafeInternal(t *testing.T) {
	st := grpcerr.Status(errors.New("pq: password auth failed for user app"), "")

	if st.Code() != codes.Internal {
		t.Errorf("code = %v, want Internal", st.Code())
	}
	if strings.Contains(st.Message(), "password auth failed") {
		t.Errorf("LEAK: raw error surfaced in status message: %q", st.Message())
	}
	if st.Message() == "" {
		t.Error("an Internal status must still carry a safe, non-empty message")
	}
}

// TestStatus_RetryInfoWhenRetryable proves a retryable error with a delay carries a
// RetryInfo detail clients can back off on; a non-retryable one does not.
func TestStatus_RetryInfoWhenRetryable(t *testing.T) {
	e := herr.New("DOWN").Kind(herr.KindUnavailable).RetryAfter(30 * time.Second)
	if ri := findRetryInfo(grpcerr.Status(e, "")); ri == nil {
		t.Fatal("expected a RetryInfo detail for a retryable error with a delay")
	} else if got := ri.GetRetryDelay().AsDuration(); got != 30*time.Second {
		t.Errorf("RetryDelay = %v, want 30s", got)
	}

	// A 404 (RetryNo) carries no RetryInfo.
	if ri := findRetryInfo(grpcerr.Status(herr.New("X").Kind(herr.KindNotFound), "")); ri != nil {
		t.Error("a non-retryable error should carry no RetryInfo")
	}
}

// findRetryInfo returns the RetryInfo detail attached to st, or nil.
func findRetryInfo(st *status.Status) *errdetails.RetryInfo {
	for _, d := range st.Details() {
		if ri, ok := d.(*errdetails.RetryInfo); ok {
			return ri
		}
	}
	return nil
}

// TestUnaryServerInterceptor_ConvertsError proves the handler one-liner: a herr error
// returned by a unary handler is converted to the mapped gRPC status, while a successful
// handler passes through untouched.
func TestUnaryServerInterceptor_ConvertsError(t *testing.T) {
	interceptor := grpcerr.UnaryServerInterceptor("")

	// Error path: herr error → mapped status.
	failing := func(context.Context, any) (any, error) {
		return nil, herr.New("NOPE").Kind(herr.KindNotFound).Public(herr.Msg("Missing."))
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, failing)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("returned error is not a gRPC status: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("code = %v, want NotFound", st.Code())
	}
	if st.Message() != "Missing." {
		t.Errorf("message = %q, want the public message", st.Message())
	}

	// Success path: response and nil error pass through unchanged.
	okHandler := func(context.Context, any) (any, error) { return "ok", nil }
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, okHandler)
	if err != nil {
		t.Errorf("success path returned error: %v", err)
	}
	if resp != "ok" {
		t.Errorf("response = %v, want it passed through unchanged", resp)
	}
}
