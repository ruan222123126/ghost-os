// API-facing bridge types used by handlers, codecs, and external clients.

package app

import (
	"ghost-os/bridge/llm"
)

const (
	actionAgentSend     = "AGENT_SEND"
	actionHumanResponse = "HUMAN_RESPONSE"
	actionConfigGet     = "CONFIG_GET"
	actionConfigUpdate  = "CONFIG_UPDATE"
	actionMemoryQuery   = "MEMORY_QUERY"
	actionMemoryArchive = "MEMORY_ARCHIVE"
)

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type agentParams struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type agentResponse struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type askHumanAwaitingResponse struct {
	Status     string `json:"status"`
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Prompt     string `json:"prompt"`
}

type humanResponseParams struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

type humanResponseAck struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Accepted   bool   `json:"accepted"`
}

type sessionIDParams struct {
	ID string `json:"id"`
}

type sessionMetadata struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	MessageCount int    `json:"message_count"`
	TokenCount   int    `json:"token_count"`
}

type sessionDetail struct {
	ID         string        `json:"id"`
	Messages   []llm.Message `json:"messages"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
	TokenCount int           `json:"token_count"`
}

type sessionDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type configResponse struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	ChatPath  string `json:"chat_path"`
	APIKeySet bool   `json:"api_key_set"`
}

type configUpdateRequest struct {
	Provider *string `json:"provider"`
	APIKey   *string `json:"api_key"`
	BaseURL  *string `json:"base_url"`
	Model    *string `json:"model"`
	ChatPath *string `json:"chat_path"`
	TraceID  string  `json:"trace_id,omitempty"`
}

type memoryTimeRangeParams struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type memoryQueryParams struct {
	SessionID string                 `json:"session_id,omitempty"`
	TimeRange *memoryTimeRangeParams `json:"time_range,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Keywords  []string               `json:"keywords,omitempty"`
	Metadata  map[string]any         `json:"metadata,omitempty"`
}

type memoryArchiveParams struct {
	SessionID string `json:"session_id"`
}

type memoryArchiveResponse struct {
	SessionID string `json:"session_id"`
	Archived  bool   `json:"archived"`
}
