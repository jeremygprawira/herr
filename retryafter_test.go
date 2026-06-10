package herr_test

import (
	"testing"
	"time"

	"github.com/jeremygeraldprawira/herr"
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
