package bus

import "encoding/json"

type RequestEnvelope struct {
	Action    string          `json:"action"`
	Params    json.RawMessage `json:"params"`
	TraceID   string          `json:"trace_id"`
	RequestID string          `json:"request_id,omitempty"`
}

type ResponseEnvelope struct {
	Status    string `json:"status"`
	Payload   any    `json:"payload"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}
