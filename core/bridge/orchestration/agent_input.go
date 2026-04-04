package orchestration

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func buildAgentUserInput(message string, images []sessionImageContent) (llm.Message, string, error) {
	text := strings.TrimSpace(message)
	content, err := normalizeAgentImages(images)
	if err != nil {
		return llm.Message{}, "", err
	}
	if text == "" && len(content) == 0 {
		return llm.Message{}, "", errAgentMessageRequired
	}
	return llm.Message{
		Role:    llm.RoleUser,
		Text:    text,
		Content: content,
	}, text, nil
}

func normalizeAgentImages(images []sessionImageContent) ([]llm.ContentPart, error) {
	if len(images) == 0 {
		return nil, nil
	}

	out := make([]llm.ContentPart, 0, len(images))
	for index, image := range images {
		part, err := normalizeAgentImage(image, index)
		if err != nil {
			return nil, err
		}
		out = append(out, part)
	}
	return out, nil
}

func normalizeAgentImage(image sessionImageContent, index int) (llm.ContentPart, error) {
	path := strings.TrimSpace(image.Path)
	url := strings.TrimSpace(image.URL)
	if path == "" && url == "" {
		return llm.ContentPart{}, fmt.Errorf("images[%d] requires path or url", index)
	}
	if image.Width < 0 || image.Height < 0 || image.Bytes < 0 {
		return llm.ContentPart{}, fmt.Errorf("images[%d] metadata must be non-negative", index)
	}

	return llm.ContentPart{
		Type: llm.ContentTypeImage,
		Image: &llm.ImageContent{
			Path:     path,
			URL:      url,
			MimeType: strings.TrimSpace(image.MimeType),
			Width:    image.Width,
			Height:   image.Height,
			SHA256:   strings.TrimSpace(image.SHA256),
			Bytes:    image.Bytes,
		},
	}, nil
}

func hasAgentInputImages(message llm.Message) bool {
	for _, part := range message.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), llm.ContentTypeImage) && part.Image != nil {
			return true
		}
	}
	return false
}
