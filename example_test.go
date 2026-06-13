package herr_test

import (
	"encoding/json"
	"fmt"

	"github.com/jeremygprawira/herr"
)

// ExampleError shows the core guarantee: an error carries rich internal detail for logs, but
// only the public surface is serialized to the client. The internal message, field, and
// wrapped cause below never appear in the JSON body.
func ExampleError() {
	e := herr.New("ORDER_NOT_FOUND").
		Kind(herr.KindNotFound).
		Public(herr.Msg("We couldn't find that order.")).
		Internal("order 4821 missing from shard eu-3"). // logs only
		With("shard", "eu-3")                           // logs only

	body, _ := json.Marshal(e) // MarshalJSON delegates to the safe wire allow-list
	fmt.Println(string(body))
	// Output: {"code":"ORDER_NOT_FOUND","message":"We couldn't find that order.","retryable":false}
}

// ExampleError_fieldErrors shows validation/field errors rendering as a typed, top-level
// errors[] array — the shape a front end maps onto individual inputs.
func ExampleError_fieldErrors() {
	e := herr.New("VALIDATION_FAILED").
		Kind(herr.KindUnprocessable).
		Public(herr.Msg("Please fix the highlighted fields.")).
		FieldError("email", "INVALID_EMAIL", "Enter a valid email address.").
		FieldError("age", "OUT_OF_RANGE", "You must be 18 or older.")

	body, _ := json.Marshal(e)
	fmt.Println(string(body))
	// Output: {"code":"VALIDATION_FAILED","message":"Please fix the highlighted fields.","retryable":false,"errors":[{"field":"email","code":"INVALID_EMAIL","message":"Enter a valid email address."},{"field":"age","code":"OUT_OF_RANGE","message":"You must be 18 or older."}]}
}
