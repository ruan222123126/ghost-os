package api

type ExternalAgentRequest struct {
	Provider       string `json:"provider,omitempty"`
	Message        string `json:"message,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Model          string `json:"model,omitempty"`
	Effort         string `json:"effort,omitempty"`
	ProjectRoot    string `json:"project_root,omitempty"`
}

type ExternalAgentStopParams struct {
	SessionID string `json:"session_id"`
}

type ExternalAgentApprovalParams struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"`
}

type ExternalAgentResponse struct {
	Status         string `json:"status"`
	Provider       string `json:"provider,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	ThreadID       string `json:"thread_id,omitempty"`
	TurnID         string `json:"turn_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

type ExternalAgentApprovalResponse struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"`
	Accepted   bool   `json:"accepted"`
}

type CodexModelCatalog struct {
	Models       []string `json:"models"`
	DefaultModel string   `json:"default_model"`
}
