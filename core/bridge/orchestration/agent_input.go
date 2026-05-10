package orchestration

import (
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
)

func buildAgentUserInput(message string, images []sessionImageContent) (llm.Message, string, error) {
	return agentturn.BuildUserInput(message, images)
}

func normalizeAgentImages(images []sessionImageContent) ([]llm.ContentPart, error) {
	if len(images) == 0 {
		return nil, nil
	}
	message, _, err := agentturn.BuildUserInput("", images)
	return message.Content, err
}

func normalizeAgentImage(image sessionImageContent, index int) (llm.ContentPart, error) {
	parts, err := normalizeAgentImages([]sessionImageContent{image})
	if err != nil {
		return llm.ContentPart{}, err
	}
	if len(parts) == 0 {
		return llm.ContentPart{}, errAgentMessageRequired
	}
	return parts[0], nil
}

func hasAgentInputImages(message llm.Message) bool {
	return agentturn.HasInputImages(message)
}
