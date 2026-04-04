package llm

import (
	"encoding/json"
	"strings"
)

func toCodexToolOutput(msg Message) string {
	baseText := strings.TrimSpace(msg.Text)
	if parsed, ok := parseCodexToolEnvelope(baseText); ok {
		switch {
		case parsed.Status == "error" && strings.TrimSpace(parsed.Error) != "":
			baseText = parsed.Error
		case strings.TrimSpace(parsed.Output) != "":
			baseText = parsed.Output
		default:
			baseText = ""
		}
	}
	if len(msg.Content) == 0 {
		return baseText
	}

	payload := codexToolOutputPayload{Text: baseText}
	for _, part := range msg.Content {
		payload.Content = append(payload.Content, toCodexToolOutputPart(part))
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return baseText
	}
	return string(encoded)
}

type codexToolOutputPayload struct {
	Text    string                `json:"text,omitempty"`
	Content []codexToolOutputPart `json:"content,omitempty"`
}

type codexToolOutputPart struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`
	Path      string `json:"path,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	ByteCount int    `json:"bytes,omitempty"`
}

func toCodexToolOutputPart(part ContentPart) codexToolOutputPart {
	entry := codexToolOutputPart{
		Type: strings.TrimSpace(part.Type),
		Text: strings.TrimSpace(part.Text),
	}
	if part.Image == nil {
		return entry
	}
	entry.Path = strings.TrimSpace(part.Image.Path)
	entry.ImageURL = strings.TrimSpace(part.Image.URL)
	entry.MimeType = strings.TrimSpace(part.Image.MimeType)
	entry.SHA256 = strings.TrimSpace(part.Image.SHA256)
	entry.ByteCount = part.Image.Bytes
	return entry
}

func parseCodexToolEnvelope(raw string) (codexToolEnvelope, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return codexToolEnvelope{}, false
	}

	var envelope codexToolEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return codexToolEnvelope{}, false
	}
	if strings.TrimSpace(envelope.Status) == "" {
		return codexToolEnvelope{}, false
	}
	return envelope, true
}
