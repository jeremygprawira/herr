package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestRetry_TriState proves the central safety property of retryability: an UNSET retry
// signal is never rendered as `false`. It is either derived from the Kind, or omitted
// entirely — so a client reading the absence of `retryable` knows "unknown", not "do not
// retry".
func TestRetry_TriState(t *testing.T) {
	cases := []struct {
		name       string
		err        *herr.Error
		wantKey    bool // should "retryable" be present at all?
		wantValue  bool // if present, its value
	}{
		{
			// KindInternal is neutral: no opinion on retryability → field omitted.
			name:    "neutral kind, unset → omitted",
			err:     herr.New("BOOM"), // defaults to KindInternal
			wantKey: false,
		},
		{
			name:      "explicit RetryYes",
			err:       herr.New("X").Retry(herr.RetryYes),
			wantKey:   true,
			wantValue: true,
		},
		{
			name:      "explicit RetryNo",
			err:       herr.New("X").Retry(herr.RetryNo),
			wantKey:   true,
			wantValue: false,
		},
		{
			name:      "derived from KindUnavailable → true",
			err:       herr.New("DOWN").Kind(herr.KindUnavailable),
			wantKey:   true,
			wantValue: true,
		},
		{
			name:      "derived from KindInvalid → false",
			err:       herr.New("BAD").Kind(herr.KindInvalid),
			wantKey:   true,
			wantValue: false,
		},
		{
			name:      "explicit overrides Kind default",
			err:       herr.New("DOWN").Kind(herr.KindUnavailable).Retry(herr.RetryNo),
			wantKey:   true,
			wantValue: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := decodeWire(t, tc.err)
			val, present := body["retryable"]
			if present != tc.wantKey {
				t.Fatalf("retryable present = %v, want %v (body=%v)", present, tc.wantKey, body)
			}
			if tc.wantKey && val != tc.wantValue {
				t.Errorf("retryable = %v, want %v", val, tc.wantValue)
			}
		})
	}
}

// TestRetryable_Accessor proves transports can read the EFFECTIVE retryability tri-state
// directly (for a gRPC RetryInfo decision, say) and that it follows the same
// convention-with-override rule as the wire `retryable` field: explicit wins, else derived
// from Kind, else RetryUnset ("unknown").
func TestRetryable_Accessor(t *testing.T) {
	if got := herr.New("BOOM").Retryable(); got != herr.RetryUnset {
		t.Errorf("neutral kind Retryable() = %v, want RetryUnset", got)
	}
	if got := herr.New("DOWN").Kind(herr.KindUnavailable).Retryable(); got != herr.RetryYes {
		t.Errorf("KindUnavailable Retryable() = %v, want RetryYes", got)
	}
	if got := herr.New("DOWN").Kind(herr.KindUnavailable).Retry(herr.RetryNo).Retryable(); got != herr.RetryNo {
		t.Errorf("explicit override Retryable() = %v, want RetryNo", got)
	}
}
