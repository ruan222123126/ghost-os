package orchestration

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams struct {
	Mode      string                `json:"mode,omitempty"`
	Message   string                `json:"message,omitempty"`
	Images    []sessionImageContent `json:"images,omitempty"`
	SessionID string                `json:"session_id,omitempty"`
}

type agentStopParams struct {
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type sessionIDParams struct {
	ID string `json:"id"`
}

type sessionGetParams struct {
	ID     string
	Limit  int
	Before *int
}

type sessionDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput
