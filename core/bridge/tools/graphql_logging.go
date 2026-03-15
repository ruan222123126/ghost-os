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
	fields := summary.FieldCount
	rootFields := summary.RootFieldCount
	fragments := summary.FragmentCount
	if err != nil {
		log.Printf(
			"trace_id=%s tool=graphql_mutation action=prepare intent_id=%s source=%s domain=%s policy=%s root_mutation=%s depth=%d fields=%d root_fields=%d fragments=%d latency_ms=%d status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(intent.IntentID),
			strings.TrimSpace(intent.Source),
			strings.TrimSpace(intent.Domain),
			strings.TrimSpace(intent.PolicyName),
			strings.TrimSpace(intent.RootMutation),
			summary.Depth,
			fields,
			rootFields,
			fragments,
			latency.Milliseconds(),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=prepare intent_id=%s source=%s domain=%s policy=%s root_mutation=%s depth=%d fields=%d root_fields=%d fragments=%d latency_ms=%d status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		summary.Depth,
		fields,
		rootFields,
		fragments,
		latency.Milliseconds(),
		status,
	)
}

func logGraphQLMutationCommit(
	traceID string,
	intent session.PendingGraphQLMutationIntent,
	responseBytes int,
	latency time.Duration,
	err error,
) {
	status := "success"
	if err != nil {
		status = "error"
		log.Printf(
			"trace_id=%s tool=graphql_mutation action=commit intent_id=%s source=%s domain=%s policy=%s root_mutation=%s approved=true response_bytes=%d latency_ms=%d status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(intent.IntentID),
			strings.TrimSpace(intent.Source),
			strings.TrimSpace(intent.Domain),
			strings.TrimSpace(intent.PolicyName),
			strings.TrimSpace(intent.RootMutation),
			responseBytes,
			latency.Milliseconds(),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=commit intent_id=%s source=%s domain=%s policy=%s root_mutation=%s approved=true response_bytes=%d latency_ms=%d status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		responseBytes,
		latency.Milliseconds(),
		status,
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
			"trace_id=%s tool=graphql_mutation action=discard intent_id=%s source=%s domain=%s policy=%s root_mutation=%s status=%s error=%q",
			strings.TrimSpace(traceID),
			strings.TrimSpace(intent.IntentID),
			strings.TrimSpace(intent.Source),
			strings.TrimSpace(intent.Domain),
			strings.TrimSpace(intent.PolicyName),
			strings.TrimSpace(intent.RootMutation),
			status,
			err.Error(),
		)
		return
	}
	log.Printf(
		"trace_id=%s tool=graphql_mutation action=discard intent_id=%s source=%s domain=%s policy=%s root_mutation=%s status=%s",
		strings.TrimSpace(traceID),
		strings.TrimSpace(intent.IntentID),
		strings.TrimSpace(intent.Source),
		strings.TrimSpace(intent.Domain),
		strings.TrimSpace(intent.PolicyName),
		strings.TrimSpace(intent.RootMutation),
		status,
	)
}
