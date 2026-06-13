package herr_test

import (
	"testing"
	"time"

	"github.com/jeremygprawira/herr"
)

// TestRetryAfter proves that a retry delay is rendered as whole seconds under `retryAfter`
// and that providing a delay implies the error IS retryable (you don't hand out a
// retry-after for something you don't want retried).
func TestRetryAfter(t *testing.T) {
	e := herr.New("DOWN").Kind(herr.KindUnavailable).RetryAfter(30 * time.Second)

	body := decodeWire(t, e)
	if body["retryAfter"] != float64(30) {
		t.Errorf("retryAfter = %v, want 30", body["retryAfter"])
	}
	if body["retryable"] != true {
		t.Errorf("retryable = %v, want true (a retry-after implies retryable)", body["retryable"])
	}
}

// TestRetryAfter_OmittedWhenUnset proves the field disappears when no delay is set, so we
// never emit a meaningless `retryAfter: 0`.
func TestRetryAfter_OmittedWhenUnset(t *testing.T) {
	e := herr.New("X").Retry(herr.RetryYes)

	body := decodeWire(t, e)
	if _, present := body["retryAfter"]; present {
		t.Errorf("retryAfter should be omitted when unset, got %v", body["retryAfter"])
	}
}

// TestRetryAfterSeconds_Accessor proves transports can read the retry delay as whole
// seconds WITHOUT decoding the wire body — the value they need to set an HTTP Retry-After
// header or a gRPC RetryInfo. Unset → 0 (no header), a sub-second delay rounds UP so it
// never reads as "no delay".
func TestRetryAfterSeconds_Accessor(t *testing.T) {
	if got := herr.New("X").RetryAfterSeconds(); got != 0 {
		t.Errorf("unset RetryAfterSeconds() = %d, want 0", got)
	}
	if got := herr.New("DOWN").Kind(herr.KindUnavailable).RetryAfter(30 * time.Second).RetryAfterSeconds(); got != 30 {
		t.Errorf("RetryAfterSeconds() = %d, want 30", got)
	}
	if got := herr.New("DOWN").RetryAfter(1500 * time.Millisecond).RetryAfterSeconds(); got != 2 {
		t.Errorf("RetryAfterSeconds() = %d, want 2 (rounded up)", got)
	}
}
