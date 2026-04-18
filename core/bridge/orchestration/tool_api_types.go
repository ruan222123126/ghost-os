package orchestration

type toolNameParams struct {
	Name string `json:"name"`
}

type toolUpdateRequest struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	PromptOverride *string `json:"prompt_override,omitempty"`
	TraceID        string  `json:"trace_id,omitempty"`
}

type toolPayload struct {
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	PromptOverride string `json:"prompt_override,omitempty"`
	InputSchema    any    `json:"input_schema,omitempty"`
}

type findIconTemplateUploadRequest struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	DataURL  string `json:"data_url"`
	TraceID  string `json:"trace_id,omitempty"`
}

type findIconTemplateUploadPayload struct {
	TemplatePath string `json:"template_path"`
	TemplateName string `json:"template_name"`
	SHA256       string `json:"sha256"`
}

type findIconPreviewRegion struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type findIconPreviewRequest struct {
	TemplatePath    string                 `json:"template_path"`
	Threshold       *float64               `json:"threshold,omitempty"`
	MaxResults      *int                   `json:"max_results,omitempty"`
	DisplayID       *int                   `json:"display_id,omitempty"`
	Region          *findIconPreviewRegion `json:"region,omitempty"`
	HoverAfterMatch bool                   `json:"hover_after_match,omitempty"`
	TraceID         string                 `json:"trace_id,omitempty"`
}

type findIconPreviewPayload struct {
	Exists     bool                   `json:"exists"`
	MatchCount int                    `json:"match_count"`
	Matches    []map[string]any       `json:"matches"`
	DisplayID  *int                   `json:"display_id,omitempty"`
	Region     *findIconPreviewRegion `json:"region,omitempty"`
	Hovered    bool                   `json:"hovered,omitempty"`
}
