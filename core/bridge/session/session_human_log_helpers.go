package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func summarizeSessionHumanLogContent(parts []llm.ContentPart) string {
	if len(parts) == 0 {
		return ""
	}
	lines := make([]string, 0, len(parts))
	for index, part := range parts {
		line, ok := summarizeSessionHumanLogContentPart(index, part)
		if !ok {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func summarizeSessionHumanLogContentPart(index int, part llm.ContentPart) (string, bool) {
	if part.Type == llm.ContentTypeImage && part.Image != nil {
		return fmt.Sprintf(
			"- [%d] image path=%q url=%q mime_type=%q width=%d height=%d bytes=%d sha256=%q",
			index,
			part.Image.Path,
			part.Image.URL,
			part.Image.MimeType,
			part.Image.Width,
			part.Image.Height,
			part.Image.Bytes,
			part.Image.SHA256,
		), true
	}
	partType := strings.TrimSpace(part.Type)
	if partType == llm.ContentTypeText || partType == "" {
		return "", false
	}
	return fmt.Sprintf("- [%d] %s", index, partType), true
}

func normalizeSessionHumanLogJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(decoded); err != nil {
		return trimmed
	}
	return strings.TrimSuffix(buffer.String(), "\n")
}

func appendSessionHumanLogBody(existingBody string, appended string) string {
	existing := strings.TrimSuffix(existingBody, "\n")
	addition := strings.TrimSpace(appended)
	switch {
	case existing == "" && addition == "":
		return ""
	case existing == "":
		return addition + "\n"
	case addition == "":
		return existing + "\n"
	default:
		return existing + "\n\n" + addition + "\n"
	}
}
