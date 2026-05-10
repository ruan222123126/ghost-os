package api

type ToolNameParams struct {
	Name string `json:"name"`
}

type ToolUpdateRequest struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	PromptOverride  *string `json:"prompt_override,omitempty"`
	SandboxMemoryMB *int    `json:"sandbox_memory_mb,omitempty"`
	TraceID         string  `json:"trace_id,omitempty"`
}

type ToolPayload struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	PromptOverride  string `json:"prompt_override,omitempty"`
	SandboxMemoryMB *int   `json:"sandbox_memory_mb,omitempty"`
	InputSchema     any    `json:"input_schema,omitempty"`
}

type FindIconTemplateUploadRequest struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	DataURL  string `json:"data_url"`
	TraceID  string `json:"trace_id,omitempty"`
}

type FindIconTemplateUploadPayload struct {
	TemplatePath string `json:"template_path"`
	TemplateName string `json:"template_name"`
	SHA256       string `json:"sha256"`
}

type FindIconPreviewRegion struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FindIconPreviewRequest struct {
	TemplatePath    string                 `json:"template_path"`
	Threshold       *float64               `json:"threshold,omitempty"`
	MaxResults      *int                   `json:"max_results,omitempty"`
	DisplayID       *int                   `json:"display_id,omitempty"`
	Region          *FindIconPreviewRegion `json:"region,omitempty"`
	HoverAfterMatch bool                   `json:"hover_after_match,omitempty"`
	TraceID         string                 `json:"trace_id,omitempty"`
}

type FindIconPreviewPayload struct {
	Exists     bool                   `json:"exists"`
	MatchCount int                    `json:"match_count"`
	Matches    []map[string]any       `json:"matches"`
	DisplayID  *int                   `json:"display_id,omitempty"`
	Region     *FindIconPreviewRegion `json:"region,omitempty"`
	Hovered    bool                   `json:"hovered,omitempty"`
}

type MousePositionRequest struct {
	TraceID string `json:"trace_id,omitempty"`
}

type MousePositionPayload struct {
	X         int      `json:"x"`
	Y         int      `json:"y"`
	DisplayID *int     `json:"display_id,omitempty"`
	ScaleX    *float64 `json:"scale_x,omitempty"`
	ScaleY    *float64 `json:"scale_y,omitempty"`
}
