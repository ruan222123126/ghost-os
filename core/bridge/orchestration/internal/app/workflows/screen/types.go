package screen

const (
	ToolID               = "screen_control"
	CoordinateRefKey     = "coordinate_ref"
	FindIconRef          = "${find_icon}"
	WorkflowStepsKey     = "workflow_steps"
	ActionKey            = "action"
	ParamsKey            = "params"
	AtomicMode           = "atomic"
	StepDelayMinMS       = 200
	StepDelayMaxMS       = 300
	FindIconDataURLParam = "workflow_template_data_url"
	LegacyDataURLParam   = "template_data_url"
	LegacyDataURLAlias   = "data_url"
	DefaultTemplateName  = "workflow-find-icon-template.png"
	LegacyDataURLMessage = "legacy find_icon data_url keys are not supported; use params.workflow_template_data_url"
)

type Step struct {
	Action     string
	ToolAction string
	Params     map[string]any
}

type TemplateUploadRequest struct {
	Filename string
	MimeType string
	DataURL  string
}

type TemplateUploadResult struct {
	TemplatePath string
	TemplateName string
}

type TemplateUploader interface {
	UploadFindIconTemplate(TemplateUploadRequest) (TemplateUploadResult, error)
}

func mapString(record map[string]any, key string) string {
	raw, ok := record[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}
