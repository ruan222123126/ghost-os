package screen

import (
	"fmt"
	"strings"

	"ghost-os/bridge/taskdefs"
)

func PrepareToolArguments(
	toolName string,
	arguments map[string]any,
	uploader TemplateUploader,
) (map[string]any, error) {
	cloned := taskdefs.CloneActionParams(arguments)
	if len(cloned) == 0 {
		return map[string]any{}, nil
	}
	if strings.TrimSpace(toolName) != ToolID {
		return cloned, nil
	}
	return prepareFindIconArguments(cloned, uploader)
}

func PrepareStepArguments(
	baseArgs map[string]any,
	step Step,
	lastFindIconOutput any,
	uploader TemplateUploader,
) (map[string]any, error) {
	args := taskdefs.CloneActionParams(baseArgs)
	if args == nil {
		args = map[string]any{}
	}
	args["mode"] = AtomicMode
	args[ActionKey] = step.ToolAction
	params, err := ResolveStepParams(step, lastFindIconOutput)
	if err != nil {
		return nil, err
	}
	args[ParamsKey] = paramsOrEmpty(params)
	return PrepareToolArguments(ToolID, args, uploader)
}

func prepareFindIconArguments(
	arguments map[string]any,
	uploader TemplateUploader,
) (map[string]any, error) {
	if !isFindIconAction(arguments) {
		return arguments, nil
	}
	rawParams, ok := arguments[ParamsKey].(map[string]any)
	if !ok {
		return arguments, nil
	}
	params, changed, err := ensureFindIconTemplatePath(rawParams, uploader)
	if err != nil {
		return nil, err
	}
	if !changed {
		return arguments, nil
	}
	arguments[ParamsKey] = params
	return arguments, nil
}

func isFindIconAction(arguments map[string]any) bool {
	action := strings.ToLower(mapString(arguments, ActionKey))
	if action != "find_icon" && action != "click_icon" {
		return false
	}
	mode := strings.ToLower(mapString(arguments, "mode"))
	return mode == "" || mode == AtomicMode
}

func paramsOrEmpty(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	return params
}

func ensureFindIconTemplatePath(
	params map[string]any,
	uploader TemplateUploader,
) (map[string]any, bool, error) {
	cloned := taskdefs.CloneActionParams(params)
	if err := rejectLegacyFindIconDataURL(cloned); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(mapString(cloned, "template_path")) != "" {
		return cloned, false, nil
	}
	req, hasUpload := buildFindIconUploadRequest(cloned)
	if !hasUpload {
		return cloned, false, nil
	}
	uploaded, err := uploadFindIconTemplate(req, uploader)
	if err != nil {
		return nil, false, err
	}
	applyFindIconUploadResult(cloned, uploaded)
	return cloned, true, nil
}

func uploadFindIconTemplate(
	req TemplateUploadRequest,
	uploader TemplateUploader,
) (TemplateUploadResult, error) {
	if uploader == nil {
		return TemplateUploadResult{}, fmt.Errorf("workflow find_icon template uploader is not configured")
	}
	return uploader.UploadFindIconTemplate(req)
}

func buildFindIconUploadRequest(params map[string]any) (TemplateUploadRequest, bool) {
	dataURL := firstNonEmptyString(params, FindIconDataURLParam)
	if dataURL == "" {
		return TemplateUploadRequest{}, false
	}
	filename := firstNonEmptyString(params, "template_filename", "filename", "template_name")
	if filename == "" {
		filename = DefaultTemplateName
	}
	return TemplateUploadRequest{
		Filename: filename,
		MimeType: findIconUploadMimeType(params, dataURL),
		DataURL:  dataURL,
	}, true
}

func findIconUploadMimeType(params map[string]any, dataURL string) string {
	if mimeType := firstNonEmptyString(params, "template_mime_type", "mime_type"); mimeType != "" {
		return mimeType
	}
	return findIconDataURLMimeType(dataURL)
}
