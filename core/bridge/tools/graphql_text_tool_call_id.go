package tools

import (
	"context"
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

func resolveGraphQLTextToolCallID(ctx context.Context) string {
	toolCallID := ToolCallIDFromContext(ctx)
	if toolCallID != "" {
		return toolCallID
	}
	return NewGraphQLTextToolCallID()
}
