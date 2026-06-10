package herr

// GRPCCode is herr's own enumeration of gRPC status codes.
//
// It exists so the CORE package can express a gRPC code WITHOUT importing the grpc module
// (keeping core dependency-free). The integer values are deliberately identical to the
// canonical google.golang.org/grpc/codes values, so the grpcerr adapter converts with a
// trivial codes.Code(c) cast. An error is never "OK", so the zero value (GRPCOK) doubles
// as the "unset → derive from Kind" sentinel.
type GRPCCode uint32

// Canonical gRPC codes (values match google.golang.org/grpc/codes).
const (
	GRPCOK                 GRPCCode = 0
	GRPCCanceled           GRPCCode = 1
	GRPCUnknown            GRPCCode = 2
	GRPCInvalidArgument    GRPCCode = 3
	GRPCDeadlineExceeded   GRPCCode = 4
	GRPCNotFound           GRPCCode = 5
	GRPCAlreadyExists      GRPCCode = 6
	GRPCPermissionDenied   GRPCCode = 7
	GRPCResourceExhausted  GRPCCode = 8
	GRPCFailedPrecondition GRPCCode = 9
	GRPCAborted            GRPCCode = 10
	GRPCOutOfRange         GRPCCode = 11
	GRPCUnimplemented      GRPCCode = 12
	GRPCInternal           GRPCCode = 13
	GRPCUnavailable        GRPCCode = 14
	GRPCDataLoss           GRPCCode = 15
	GRPCUnauthenticated    GRPCCode = 16
)

// kindGRPC maps each Kind to its default gRPC code (the single source of truth for the
// gRPC side of the convention).
var kindGRPC = map[Kind]GRPCCode{
	KindInternal:     GRPCInternal,
	KindInvalid:      GRPCInvalidArgument,
	KindUnauthorized: GRPCUnauthenticated,
	KindForbidden:    GRPCPermissionDenied,
	KindNotFound:     GRPCNotFound,
	KindConflict:     GRPCAborted,
	KindRateLimited:  GRPCResourceExhausted,
	KindTimeout:      GRPCDeadlineExceeded,
	KindUnavailable:  GRPCUnavailable,
}

// kindWS maps each Kind to a WebSocket close code (RFC 6455). 1008 = Policy Violation,
// 1011 = Internal Error, 1013 = Try Again Later.
var kindWS = map[Kind]int{
	KindInternal:     1011,
	KindInvalid:      1008,
	KindUnauthorized: 1008,
	KindForbidden:    1008,
	KindNotFound:     1008,
	KindConflict:     1008,
	KindRateLimited:  1013,
	KindTimeout:      1011,
	KindUnavailable:  1013,
}

// GRPC sets an EXPLICIT gRPC code override and returns the receiver for chaining.
func (e *Error) GRPC(c GRPCCode) *Error {
	if e == nil {
		return nil
	}
	e.grpcCode = c
	return e
}

// GRPCCode resolves the effective gRPC code: explicit override (non-OK) wins, else the
// Kind default, else GRPCInternal as the safe floor.
func (e *Error) GRPCCode() GRPCCode {
	if e == nil {
		return GRPCInternal
	}
	if e.grpcCode != GRPCOK {
		return e.grpcCode
	}
	if c, ok := kindGRPC[e.kind]; ok {
		return c
	}
	return GRPCInternal
}

// WS sets an EXPLICIT WebSocket close-code override and returns the receiver for chaining.
func (e *Error) WS(closeCode int) *Error {
	if e == nil {
		return nil
	}
	e.wsClose = closeCode
	return e
}

// WSClose resolves the effective WebSocket close code: explicit override wins, else the
// Kind default, else 1011 (Internal Error) as the safe floor.
func (e *Error) WSClose() int {
	if e == nil {
		return 1011
	}
	if e.wsClose != 0 {
		return e.wsClose
	}
	if c, ok := kindWS[e.kind]; ok {
		return c
	}
	return 1011
}
