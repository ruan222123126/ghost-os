package tools

import (
	"strings"
	"time"

	"ghost-os/bridge/session"
)

func buildGraphQLMutationReceipt(
	intent session.PendingGraphQLMutationIntent,
	attempt int,
	state string,
	startedAt time.Time,
	finishedAt time.Time,
	body []byte,
	httpStatus int,
	err error,
) session.GraphQLMutationReceipt {
	receipt := session.GraphQLMutationReceipt{
		Attempt:       attempt,
		State:         strings.TrimSpace(state),
		StartedAt:     startedAt.UTC(),
		FinishedAt:    finishedAt.UTC(),
		DeliveryKey:   strings.TrimSpace(intent.DeliveryKey),
		RequestHash:   strings.TrimSpace(intent.RequestHash),
		ResponseBytes: len(body),
		HTTPStatus:    httpStatus,
	}
	if len(body) > 0 {
		receipt.ResponseHash = hashGraphQLMutationBytes(body)
	}
	if err != nil {
		receipt.Error = err.Error()
	}
	return receipt
}

func applyGraphQLMutationReceipt(
	intent session.PendingGraphQLMutationIntent,
	state string,
	receipt session.GraphQLMutationReceipt,
) session.PendingGraphQLMutationIntent {
	intent.Status = strings.TrimSpace(state)
	intent.CommitState = intent.Status
	intent.AttemptCount = receipt.Attempt
	intent.LastAttemptAt = receipt.FinishedAt
	intent.LastError = strings.TrimSpace(receipt.Error)
	intent.ResponseHash = strings.TrimSpace(receipt.ResponseHash)
	intent.ResponseBytes = receipt.ResponseBytes
	intent.Receipts = append(intent.Receipts, receipt)
	if intent.Status == session.GraphQLMutationIntentExecuted && intent.ExecutedAt.IsZero() {
		intent.ExecutedAt = receipt.FinishedAt.UTC()
	}
	return intent
}

func lastGraphQLMutationReceipt(
	receipts []session.GraphQLMutationReceipt,
) *session.GraphQLMutationReceipt {
	if len(receipts) == 0 {
		return nil
	}
	receipt := receipts[len(receipts)-1]
	return &receipt
}
