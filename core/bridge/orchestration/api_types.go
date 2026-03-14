package orchestration

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type agentStopParams struct {
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type sessionIDParams struct {
	ID string `json:"id"`
}

type sessionDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput
