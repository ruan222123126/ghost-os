package tools

import (
	"log"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

func logGraphQLQuery(
	traceID string,
	source string,
	domain string,
	summary graphQLQuerySummary,
	responseBytes int,
	latency time.Duration,
	err error,
) {
	status := "success"
	if err != nil {
		status = "error"
		log.Printf(
			"trace_id=%s tool=graphql_query source=%s domain=%s operation=%s depth=%d fields=%d root_fields=%d fragments=%d response_bytes=%d latency_ms=%d status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(source),
			strings.TrimSpace(domain),
			graphQLQueryOperationLabel(summary),
			summary.Depth,
			summary.FieldCount,
			summary.RootFieldCount,
			summary.FragmentCount,
			responseBytes,
			latency.Milliseconds(),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_query source=%s domain=%s operation=%s depth=%d fields=%d root_fields=%d fragments=%d response_bytes=%d latency_ms=%d status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(source),
		strings.TrimSpace(domain),
		graphQLQueryOperationLabel(summary),
		summary.Depth,
		summary.FieldCount,
		summary.RootFieldCount,
		summary.FragmentCount,
		responseBytes,
		latency.Milliseconds(),
		status,
	)
}

func logGraphQLSchemaLookup(
	traceID string,
	source string,
	domain string,
	action string,
	name string,
	err error,
) {
	status := "success"
	if err != nil {
		status = "error"
		log.Printf(
			"trace_id=%s tool=graphql_schema_lookup source=%s domain=%s action=%s name=%s status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(source),
			strings.TrimSpace(domain),
			strings.TrimSpace(action),
			strings.TrimSpace(name),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_schema_lookup source=%s domain=%s action=%s name=%s status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(source),
		strings.TrimSpace(domain),
		strings.TrimSpace(action),
		strings.TrimSpace(name),
		status,
	)
}

func graphQLQueryOperationLabel(summary graphQLQuerySummary) string {
	if strings.TrimSpace(summary.OperationName) == "" {
		return "anonymous"
	}
	return strings.TrimSpace(summary.OperationName)
}

func logGraphQLMutationPrepare(
	traceID string,
	intent session.PendingGraphQLMutationIntent,
	summary graphQLOperationSummary,
	latency time.Duration,
	err error,
) {
	status := "success"
	if err != nil {
		status = "error"
	}
	if err != nil {
		log.Printf(
			"trace_id=%s tool=graphql_mutation action=prepare intent_id=%s source=%s domain=%s policy=%s root_mutation=%s delivery_key=%s request_hash=%s depth=%d fields=%d root_fields=%d fragments=%d latency_ms=%d status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(intent.IntentID),
			strings.TrimSpace(intent.Source),
			strings.TrimSpace(intent.Domain),
			strings.TrimSpace(intent.PolicyName),
			strings.TrimSpace(intent.RootMutation),
			strings.TrimSpace(intent.DeliveryKey),
			strings.TrimSpace(intent.RequestHash),
			summary.Depth,
			summary.FieldCount,
			summary.RootFieldCount,
			summary.FragmentCount,
			latency.Milliseconds(),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=prepare intent_id=%s source=%s domain=%s policy=%s root_mutation=%s delivery_key=%s request_hash=%s depth=%d fields=%d root_fields=%d fragments=%d latency_ms=%d status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		strings.TrimSpace(intent.DeliveryKey),
		strings.TrimSpace(intent.RequestHash),
		summary.Depth,
		summary.FieldCount,
		summary.RootFieldCount,
		summary.FragmentCount,
		latency.Milliseconds(),
		status,
	)
}

func logGraphQLMutationCommit(
	traceID string,
	intent session.PendingGraphQLMutationIntent,
	receipt *session.GraphQLMutationReceipt,
	latency time.Duration,
	err error,
) {
	status := "success"
	commitState := strings.TrimSpace(intent.CommitState)
	attempt := intent.AttemptCount
	responseHash := strings.TrimSpace(intent.ResponseHash)
	responseBytes := intent.ResponseBytes
	httpStatus := 0
	if receipt != nil {
		commitState = strings.TrimSpace(receipt.State)
		attempt = receipt.Attempt
		responseHash = strings.TrimSpace(receipt.ResponseHash)
		responseBytes = receipt.ResponseBytes
		httpStatus = receipt.HTTPStatus
	}
	if err != nil {
		status = "error"
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=commit intent_id=%s source=%s domain=%s policy=%s root_mutation=%s commit_state=%s delivery_key=%s attempt=%d request_hash=%s response_hash=%s response_bytes=%d http_status=%d latency_ms=%d status=%s error=%q",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		commitState,
		strings.TrimSpace(intent.DeliveryKey),
		attempt,
		strings.TrimSpace(intent.RequestHash),
		responseHash,
		responseBytes,
		httpStatus,
		latency.Milliseconds(),
		status,
		errorString(err),
	)
}

func logGraphQLMutationDiscard(
	traceID string,
	intent session.PendingGraphQLMutationIntent,
	err error,
) {
	status := "discarded"
	if err != nil {
		status = "error"
		log.Printf(
			"trace_id=%s tool=graphql_mutation action=discard intent_id=%s source=%s domain=%s policy=%s root_mutation=%s commit_state=%s delivery_key=%s status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(intent.IntentID),
			strings.TrimSpace(intent.Source),
			strings.TrimSpace(intent.Domain),
			strings.TrimSpace(intent.PolicyName),
			strings.TrimSpace(intent.RootMutation),
			strings.TrimSpace(intent.CommitState),
			strings.TrimSpace(intent.DeliveryKey),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=discard intent_id=%s source=%s domain=%s policy=%s root_mutation=%s commit_state=%s delivery_key=%s status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		strings.TrimSpace(intent.CommitState),
		strings.TrimSpace(intent.DeliveryKey),
		status,
	)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
