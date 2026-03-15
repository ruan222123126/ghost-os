package session

import "time"

type GraphQLMutationReceipt struct {
	Attempt       int       `json:"attempt"`
	State         string    `json:"state"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DeliveryKey   string    `json:"delivery_key,omitempty"`
	RequestHash   string    `json:"request_hash,omitempty"`
	ResponseHash  string    `json:"response_hash,omitempty"`
	ResponseBytes int       `json:"response_bytes,omitempty"`
	HTTPStatus    int       `json:"http_status,omitempty"`
	Error         string    `json:"error,omitempty"`
}

func cloneGraphQLMutationReceipts(
	raw []GraphQLMutationReceipt,
) []GraphQLMutationReceipt {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLMutationReceipt, 0, len(raw))
	for _, receipt := range raw {
		out = append(out, GraphQLMutationReceipt{
			Attempt:       receipt.Attempt,
			State:         receipt.State,
			StartedAt:     receipt.StartedAt.UTC(),
			FinishedAt:    receipt.FinishedAt.UTC(),
			DeliveryKey:   receipt.DeliveryKey,
			RequestHash:   receipt.RequestHash,
			ResponseHash:  receipt.ResponseHash,
			ResponseBytes: receipt.ResponseBytes,
			HTTPStatus:    receipt.HTTPStatus,
			Error:         receipt.Error,
		})
	}
	return out
}
