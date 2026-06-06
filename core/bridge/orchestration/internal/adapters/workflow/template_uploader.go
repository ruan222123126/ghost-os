package workflow

import (
	"ghost-os/bridge/orchestration/internal/app/tools"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/orchestration/internal/contracts/api"
)

type TemplateUploader struct{}

func (TemplateUploader) UploadFindIconTemplate(
	req appworkflows.TemplateUploadRequest,
) (appworkflows.TemplateUploadResult, error) {
	payload, err := tools.ExecuteFindIconTemplateUpload(api.FindIconTemplateUploadRequest{
		Filename: req.Filename,
		MimeType: req.MimeType,
		DataURL:  req.DataURL,
	})
	if err != nil {
		return appworkflows.TemplateUploadResult{}, err
	}
	return appworkflows.TemplateUploadResult{
		TemplatePath: payload.TemplatePath,
		TemplateName: payload.TemplateName,
	}, nil
}
