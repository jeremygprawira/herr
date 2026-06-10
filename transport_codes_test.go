package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestGRPCMapping proves Kind derives a sensible gRPC status code, with an explicit
// override winning. The codes are herr's own dependency-free enum (numerically identical
// to canonical gRPC codes) so the core package never imports the grpc module.
func TestGRPCMapping(t *testing.T) {
	if got := herr.New("X").Kind(herr.KindNotFound).GRPCCode(); got != herr.GRPCNotFound {
		t.Errorf("NotFound → %v, want GRPCNotFound", got)
	}
	if got := herr.New("X").Kind(herr.KindRateLimited).GRPCCode(); got != herr.GRPCResourceExhausted {
		t.Errorf("RateLimited → %v, want GRPCResourceExhausted", got)
	}
	// Unclassified defaults to Internal.
	if got := herr.New("X").GRPCCode(); got != herr.GRPCInternal {
		t.Errorf("default → %v, want GRPCInternal", got)
	}
	// Explicit override wins.
	if got := herr.New("X").Kind(herr.KindNotFound).GRPC(herr.GRPCAborted).GRPCCode(); got != herr.GRPCAborted {
		t.Errorf("override → %v, want GRPCAborted", got)
	}
}

// TestWSMapping proves Kind derives a WebSocket close code, with an explicit override
// winning.
func TestWSMapping(t *testing.T) {
	if got := herr.New("X").Kind(herr.KindRateLimited).WSClose(); got != 1013 { // Try Again Later
		t.Errorf("RateLimited → %d, want 1013", got)
	}
	if got := herr.New("X").Kind(herr.KindInvalid).WSClose(); got != 1008 { // Policy Violation
		t.Errorf("Invalid → %d, want 1008", got)
	}
	// Unclassified defaults to 1011 (Internal Error).
	if got := herr.New("X").WSClose(); got != 1011 {
		t.Errorf("default → %d, want 1011", got)
	}
	// Explicit override wins.
	if got := herr.New("X").WS(1000).WSClose(); got != 1000 {
		t.Errorf("override → %d, want 1000", got)
	}
}
