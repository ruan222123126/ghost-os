package api

type SessionImageContent struct {
	Path     string `json:"path,omitempty"`
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Bytes    int    `json:"bytes,omitempty"`
}

type SessionFileContent struct {
	ArtifactID  string `json:"artifact_id"`
	Name        string `json:"name"`
	MimeType    string `json:"mime_type,omitempty"`
	Bytes       int    `json:"bytes,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	DownloadURL string `json:"download_url"`
	SourcePath  string `json:"source_path,omitempty"`
	Note        string `json:"note,omitempty"`
}

type SessionContentPart struct {
	Type  string               `json:"type"`
	Text  string               `json:"text,omitempty"`
	Image *SessionImageContent `json:"image,omitempty"`
	File  *SessionFileContent  `json:"file,omitempty"`
}

type SessionToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type SessionToolResult struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id,omitempty"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

type SessionHumanInteraction struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
	Answer        string           `json:"answer,omitempty"`
}

type SessionMessage struct {
	Index            int                      `json:"index"`
	Role             string                   `json:"role"`
	Text             string                   `json:"text,omitempty"`
	Content          []SessionContentPart     `json:"content,omitempty"`
	ToolCalls        []SessionToolCall        `json:"tool_calls,omitempty"`
	ToolResult       *SessionToolResult       `json:"tool_result,omitempty"`
	HumanInteraction *SessionHumanInteraction `json:"human_interaction,omitempty"`
	ToolCallID       string                   `json:"tool_call_id,omitempty"`
	InProgress       bool                     `json:"in_progress,omitempty"`
	Thinking         string                   `json:"thinking,omitempty"`
}

type SessionMetadata struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	MessageCount int    `json:"message_count"`
	TokenCount   int    `json:"token_count"`
}

type SessionRuntimeSelection struct {
	Runtime      string `json:"runtime"`
	Provider     string `json:"provider,omitempty"`
	ProviderType string `json:"provider_type,omitempty"`
	Model        string `json:"model,omitempty"`
	Mode         string `json:"mode,omitempty"`
}

type SessionMessagePage struct {
	Limit         int  `json:"limit"`
	Before        *int `json:"before,omitempty"`
	StartIndex    *int `json:"start_index,omitempty"`
	EndIndex      *int `json:"end_index,omitempty"`
	HasMoreBefore bool `json:"has_more_before"`
	NextBefore    *int `json:"next_before,omitempty"`
}

type SessionTurnDraftSegment struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type SessionTurnDraftTool struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolStatus string `json:"tool_status,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

type SessionTurnDraftPendingQuestion struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

type SessionTurnDraft struct {
	TraceID           string                            `json:"trace_id"`
	Turn              int                               `json:"turn"`
	Status            string                            `json:"status"`
	Error             string                            `json:"error,omitempty"`
	PendingQuestions  []SessionTurnDraftPendingQuestion `json:"pending_questions"`
	AssistantSegments []SessionTurnDraftSegment         `json:"assistant_segments"`
	ThinkingSegments  []SessionTurnDraftSegment         `json:"thinking_segments"`
	Tools             []SessionTurnDraftTool            `json:"tools"`
	ItemOrder         []string                          `json:"item_order"`
}

type SessionDetail struct {
	ID                   string                   `json:"id"`
	Title                string                   `json:"title"`
	Messages             []SessionMessage         `json:"messages"`
	CreatedAt            string                   `json:"created_at"`
	UpdatedAt            string                   `json:"updated_at"`
	MessageCount         int                      `json:"message_count"`
	Page                 SessionMessagePage       `json:"page"`
	TokenCount           int                      `json:"token_count"`
	TurnDraft            *SessionTurnDraft        `json:"turn_draft,omitempty"`
	LastRuntimeSelection *SessionRuntimeSelection `json:"last_runtime_selection,omitempty"`
}

type SessionAppendMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type SessionAppendRequest struct {
	SessionID    string                 `json:"session_id"`
	ExpectedHead *int                   `json:"expected_head,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Messages     []SessionAppendMessage `json:"messages"`
	TraceID      string                 `json:"trace_id,omitempty"`
}

type SessionAppendResponse struct {
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
	MessageCount int    `json:"message_count"`
	UpdatedAt    string `json:"updated_at"`
}
