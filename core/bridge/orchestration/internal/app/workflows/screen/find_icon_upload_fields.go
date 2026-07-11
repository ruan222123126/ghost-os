package screen

import (
	"errors"
	"strings"
)

func applyFindIconUploadResult(params map[string]any, uploaded TemplateUploadResult) {
	params["template_path"] = uploaded.TemplatePath
	if strings.TrimSpace(mapString(params, "template_name")) == "" {
		params["template_name"] = uploaded.TemplateName
	}
	removeFindIconUploadFields(params)
}

func rejectLegacyFindIconDataURL(params map[string]any) error {
	if strings.TrimSpace(mapString(params, LegacyDataURLParam)) != "" {
		return errors.New(LegacyDataURLMessage)
	}
	if strings.TrimSpace(mapString(params, LegacyDataURLAlias)) != "" {
		return errors.New(LegacyDataURLMessage)
	}
	return nil
}

func findIconDataURLMimeType(dataURL string) string {
	if !strings.HasPrefix(dataURL, "data:") {
		return ""
	}
	prefix, _, found := strings.Cut(dataURL, ",")
	if !found {
		return ""
	}
	raw := strings.TrimPrefix(prefix, "data:")
	raw = strings.TrimSuffix(raw, ";base64")
	return strings.ToLower(strings.TrimSpace(raw))
}

func removeFindIconUploadFields(params map[string]any) {
	delete(params, FindIconDataURLParam)
	delete(params, LegacyDataURLParam)
	delete(params, "template_filename")
	delete(params, "template_mime_type")
	delete(params, LegacyDataURLAlias)
	delete(params, "filename")
	delete(params, "mime_type")
}

func firstNonEmptyString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(mapString(record, key))
		if value != "" {
			return value
		}
	}
	return ""
}
