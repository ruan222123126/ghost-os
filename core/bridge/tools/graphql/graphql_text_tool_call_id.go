package graphql

import (
	"fmt"
	"sync/atomic"
	"time"
)

const graphQLTextToolCallPrefix = "graphql-text-call"

var graphQLTextToolCallSequence atomic.Uint64

func NewGraphQLTextToolCallID() string {
	sequence := graphQLTextToolCallSequence.Add(1)
	return fmt.Sprintf(
		"%s-%d-%d",
		graphQLTextToolCallPrefix,
		time.Now().UTC().UnixNano(),
		sequence,
	)
}
