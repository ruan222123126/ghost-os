package workflows

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func ensureFindIconTemplatePath(
	params map[string]any,
	uploader TemplateUploader,
) (map[string]any, bool, error) {
	cloned := bridgeTasks.CloneActionParams(params)
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
		filename = FindIconDefaultTemplateName
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
