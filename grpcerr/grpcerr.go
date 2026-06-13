// Package grpcerr is the gRPC transport adapter for herr.
//
// A gRPC handler returns/produces a *herr.Error (or any error); grpcerr maps it to a
// *status.Status whose code follows the error's Kind and whose message is the SAFE public
// message. When the error advertises retryability with a delay, a RetryInfo detail is
// attached so clients can back off. Because it renders only through the core's safe public
// surface, no internal detail (internal message, fields, cause, stack) can reach the status.
//
// It is its OWN Go module so importing it never pulls grpc into a project that only wants
// the dependency-free herr core.
package grpcerr

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/jeremygeraldprawira/herr"
)

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that maps any error a unary
// handler returns into the safe gRPC status (via Status), leaving successful responses
// untouched. A handler can therefore just `return someHerrError` and the client receives the
// correct code, public message, and RetryInfo — no per-handler conversion boilerplate.
//
// The locale is fixed here for simplicity; resolving it per-call from incoming metadata is a
// follow-up that can wrap this same mapping.
func UnaryServerInterceptor(locale string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, Status(err, locale).Err()
		}
		return resp, nil
	}
}

// Status maps err to a gRPC *status.Status.
//
// Flow: coerce any error to a *herr.Error (a non-herr error becomes a safe Internal status
// wrapping the cause, which stays server-side), translate the Kind-derived gRPC code, use
// the localized public message as the status message, and attach RetryInfo when the error
// advertises a retry delay.
func Status(err error, locale string) *status.Status {
	he := coerce(err)

	// herr.GRPCCode values are defined to equal google.golang.org/grpc/codes values, so the
	// conversion is a direct cast — no lookup table, no drift.
	code := codes.Code(uint32(he.GRPCCode()))
	st := status.New(code, publicMessage(he, locale))

	if detail := retryInfo(he); detail != nil {
		if withDetail, derr := st.WithDetails(detail); derr == nil {
			st = withDetail
		}
		// If attaching the detail fails (it shouldn't for a well-formed message), we keep
		// the plain status rather than dropping the whole response.
	}
	return st
}

// coerce returns err as a *herr.Error, wrapping a non-herr error as a server fault so its
// detail stays server-side (available to logs) and never reaches the status sent to a client.
func coerce(err error) *herr.Error {
	var he *herr.Error
	if errors.As(err, &he) {
		return he
	}
	return herr.New("INTERNAL").Kind(herr.KindInternal).Wrap(err)
}

// publicMessage extracts the safe, localized public message by rendering the error's wire
// body (the same allow-listed DTO every transport uses) and reading the message field —
// keeping grpcerr on the safe side of the public/internal split.
func publicMessage(he *herr.Error, locale string) string {
	raw, err := json.Marshal(he.Body(locale))
	if err != nil {
		return ""
	}
	var body struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &body)
	return body.Message
}

// retryInfo builds a RetryInfo detail when the error advertises retryability WITH a concrete
// delay, and nil otherwise. A retry signal without a delay carries no RetryInfo (there is no
// number to advertise); an explicit RetryNo or unknown never gets one.
func retryInfo(he *herr.Error) *errdetails.RetryInfo {
	if he.Retryable() != herr.RetryYes {
		return nil
	}
	secs := he.RetryAfterSeconds()
	if secs <= 0 {
		return nil
	}
	return &errdetails.RetryInfo{
		RetryDelay: durationpb.New(time.Duration(secs) * time.Second),
	}
}
