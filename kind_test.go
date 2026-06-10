package herr_test

import (
	"testing"

	"github.com/jeremygeraldprawira/herr"
)

// TestKind_DrivesDefaultHTTPStatus proves the "convention with override" rule for the
// HTTP status: you classify the error once by Kind, and the correct status falls out
// automatically — but you can always override it explicitly.
func TestKind_DrivesDefaultHTTPStatus(t *testing.T) {
	cases := []struct {
		name string
		err  *herr.Error
		want int
	}{
		{
			// No Kind set → defaults to Internal → 500. This is the safe floor: an
			// unclassified error is treated as a server fault, never a 200.
			name: "unclassified defaults to 500",
			err:  herr.New("BOOM"),
			want: 500,
		},
		{
			name: "not found → 404",
			err:  herr.New("MISSING").Kind(herr.KindNotFound),
			want: 404,
		},
		{
			name: "invalid → 400",
			err:  herr.New("BAD_INPUT").Kind(herr.KindInvalid),
			want: 400,
		},
		{
			name: "unavailable → 503",
			err:  herr.New("DOWN").Kind(herr.KindUnavailable),
			want: 503,
		},
		{
			// Explicit override wins over the Kind-derived default.
			name: "explicit Status overrides the Kind default",
			err:  herr.New("MISSING").Kind(herr.KindNotFound).Status(418),
			want: 418,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.HTTPStatus(); got != tc.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tc.want)
			}
		})
	}
}
