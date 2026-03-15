package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GraphQLMutationTool struct {
	httpClient *http.Client
	registry   *GraphQLSourceRegistry
}

func NewGraphQLMutationTool(registry *GraphQLSourceRegistry) Tool {
	return &GraphQLMutationTool{
		httpClient: &http.Client{},
		registry:   registry,
	}
}

func (GraphQLMutationTool) Name() string {
	return "graphql_mutation"
}

func (GraphQLMutationTool) Description() string {
	return "Prepare, commit, discard, or inspect explicitly approved GraphQL mutations. Review the available mutation policy metadata before prepare when it is available."
}

func (GraphQLMutationTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["prepare","commit","retry_commit","status","discard","list_pending"]},
			"source":{"type":"string","description":"GraphQL source name for action=prepare."},
			"domain":{"type":"string","description":"Domain name for action=prepare. Required because mutation policies are domain-scoped."},
			"mutation":{"type":"string","description":"GraphQL mutation document for action=prepare."},
			"variables":{"type":"object","description":"Optional GraphQL variables object for action=prepare."},
			"operation_name":{"type":"string","description":"Optional operation name when the document contains multiple mutation operations."},
			"intent_id":{"type":"string","description":"Frozen mutation intent id for action=commit, retry_commit, status, or discard."}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t *GraphQLMutationTool) InterpretResult(output string) ExecuteMeta {
	return interpretGraphQLMutationAwaitingResult(output)
}

func (t *GraphQLMutationTool) Execute(
	ctx context.Context,
	argsJSON json.RawMessage,
	traceID string,
) (string, error) {
	args, err := decodeGraphQLMutationArgs(argsJSON)
	if err != nil {
		return "", err
	}
	switch args.Action {
	case graphQLMutationActionPrepare:
		return t.prepare(ctx, args, traceID)
	case graphQLMutationActionCommit:
		return t.commit(ctx, args, traceID)
	case graphQLMutationActionRetryCommit:
		return t.retryCommit(ctx, args, traceID)
	case graphQLMutationActionStatus:
		return t.status(ctx, args)
	case graphQLMutationActionDiscard:
		return t.discard(ctx, args, traceID)
	case graphQLMutationActionListPending:
		return t.listPending(ctx)
	default:
		return "", fmt.Errorf("action must be one of: prepare, commit, retry_commit, status, discard, list_pending")
	}
}
