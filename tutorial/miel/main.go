package main

import (
	"context"
	miel "github.com/worldiety/mistral/lib/go/dsl/v1"
)

// Request is an arbitrary defined input parameter type parsed from an application/json body.
// See also lines 41 and 42 for deserialization.
type Request struct {
	Portfolio miel.UUIDs         `json:"device-ids"`
	Metric    miel.UUID          `json:"metric-id"`
	Type      miel.AggregateFunc `json:"type"`
	Range     miel.Range
	TZ        miel.TZ
}

// Response is an arbitrary defined output parameter type serialized as an application/json body.
type Response struct {
	Portfolio     miel.FPoints
	MyDevices     miel.FGroup
	MyDeviceNames []string
}

// Declare is a function which serves two purposes:
//  1. declare which types are input and output parameters. This is best-practice to generate automatic documentation.
//  2. return examples for each, also just for automatic documentation.
// See also line 58.
func Declare() (interface{}, interface{}) {
	return Request{
		Portfolio: miel.UUIDs{miel.UUID{}},
		Type:      miel.AvgY,
	}, Response{}
}

// Eval provides the actual calculation kernel and operation. It extracts concrete instances from the given context.
// The implementation must be thread-safe and must not share any state between executions.
// It is undefined, whether Eval is executed serializable, concurrently and/or on multiple independent Mistral cluster
// instances at the same time.
// See also line 59.
func Eval(ctx context.Context) {
	var request Request         // declare a variable using our custom request type
	miel.Request(ctx, &request) // parse our custom request type

	loc := request.TZ.MustParse()
	miel.Query(ctx).
		FindInRange(request.Portfolio, request.Metric, request.Range).
		ForEach(func(pts miel.Points) miel.Points {
			return pts.GroupByDay(miel.NoDrift, miel.AlignGroupStart, loc).Reduce(miel.AvgY)
		})

	miel.Response(ctx, Response{})
}

// main provides the default launching point.
func main() {
	miel.Configure().
		Parameter(Declare).
		Start(Eval) // eventually execute the Eval function
}
