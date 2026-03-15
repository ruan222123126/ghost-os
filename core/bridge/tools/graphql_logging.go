package tools

import (
	"log"
	"strings"
	"time"
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
