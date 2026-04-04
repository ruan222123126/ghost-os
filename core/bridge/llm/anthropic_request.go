package llm

import (
	"fmt"
	"strings"
)

// toAnthropicRequest 把统一请求转换为 Anthropic messages/tool_use 协议。
func toAnthropicRequest(model string, maxTokens int, request CompletionRequest) (anthropicRequest, error) {
	if err := validateRequestMessageToolProtocol(request.Messages); err != nil {
		return anthropicRequest{}, err
	}

	messages, systemParts, err := toAnthropicMessages(request.Messages)
	if err != nil {
		return anthropicRequest{}, err
	}
	tools, err := toAnthropicTools(request.Tools)
	if err != nil {
		return anthropicRequest{}, err
	}

	out := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  messages,
		Tools:     tools,
	}
	if len(systemParts) > 0 {
		out.System = strings.Join(systemParts, "\n\n")
	}
	return out, nil
}

func toAnthropicMessages(messages []Message) ([]anthropicMessage, []string, error) {
	converted := make([]anthropicMessage, 0, len(messages))
	systemParts := make([]string, 0, 2)
	for _, msg := range messages {
		projected, err := toAnthropicMessageProjection(msg)
		if err != nil {
			return nil, nil, err
		}
		if projected.system != "" {
			systemParts = append(systemParts, projected.system)
		}
		if projected.hasMessage {
			converted = append(converted, projected.message)
		}
	}
	return converted, systemParts, nil
}

type anthropicMessageProjection struct {
	message    anthropicMessage
	hasMessage bool
	system     string
}

func toAnthropicMessageProjection(msg Message) (anthropicMessageProjection, error) {
	switch msg.Role {
	case RoleSystem:
		return anthropicMessageProjection{
			system: strings.TrimSpace(msg.Text),
		}, nil
	case RoleUser:
		message, err := toAnthropicUserMessage(msg)
		if err != nil {
			return anthropicMessageProjection{}, err
		}
		return anthropicMessageProjection{message: message, hasMessage: true}, nil
	case RoleAssistant:
		message, err := toAnthropicAssistantMessage(msg)
		if err != nil {
			return anthropicMessageProjection{}, err
		}
		return anthropicMessageProjection{message: message, hasMessage: true}, nil
	case RoleTool:
		message, err := toAnthropicToolMessage(msg)
		if err != nil {
			return anthropicMessageProjection{}, err
		}
		return anthropicMessageProjection{message: message, hasMessage: true}, nil
	default:
		return anthropicMessageProjection{}, fmt.Errorf("unsupported message role for anthropic: %q", msg.Role)
	}
}

func toAnthropicUserMessage(msg Message) (anthropicMessage, error) {
	content, err := toAnthropicTextImageContent(msg)
	if err != nil {
		return anthropicMessage{}, err
	}
	return anthropicMessage{
		Role:    "user",
		Content: content,
	}, nil
}

func toAnthropicToolMessage(msg Message) (anthropicMessage, error) {
	content, err := toAnthropicTextImageContent(msg)
	if err != nil {
		return anthropicMessage{}, err
	}
	return anthropicMessage{
		Role: "user",
		Content: []anthropicContentBlock{
			{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   content,
			},
		},
	}, nil
}

func toAnthropicTools(tools []ToolDef) ([]anthropicTool, error) {
	converted := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		schema, err := decodeJSONObject(tool.Parameters)
		if err != nil {
			return nil, fmt.Errorf("decode schema for tool %q: %w", tool.Name, err)
		}
		converted = append(converted, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}
	return converted, nil
}

// toAnthropicAssistantMessage 将 assistant 文本与 tool calls 编码为 content blocks。
func toAnthropicAssistantMessage(msg Message) (anthropicMessage, error) {
	blocks := make([]anthropicContentBlock, 0, len(msg.ToolCalls)+1)
	text := strings.TrimSpace(msg.Text)
	if text != "" {
		blocks = append(blocks, anthropicContentBlock{
			Type: "text",
			Text: text,
		})
	}
	for _, call := range msg.ToolCalls {
		input, err := decodeJSONObject(call.Arguments)
		if err != nil {
			return anthropicMessage{}, fmt.Errorf("decode arguments for tool %q: %w", call.Name, err)
		}
		blocks = append(blocks, anthropicContentBlock{
			Type:  "tool_use",
			ID:    call.ID,
			Name:  call.Name,
			Input: input,
		})
	}
	if len(blocks) == 0 {
		return anthropicMessage{Role: "assistant", Content: ""}, nil
	}
	if len(msg.ToolCalls) == 0 && len(blocks) == 1 && blocks[0].Type == "text" {
		return anthropicMessage{Role: "assistant", Content: blocks[0].Text}, nil
	}
	return anthropicMessage{
		Role:    "assistant",
		Content: blocks,
	}, nil
}

func toAnthropicTextImageContent(msg Message) (any, error) {
	if len(msg.Content) == 0 {
		return msg.Text, nil
	}

	blocks := make([]anthropicContentBlock, 0, len(msg.Content)+1)
	text := strings.TrimSpace(msg.Text)
	if text != "" {
		blocks = append(blocks, anthropicContentBlock{
			Type: "text",
			Text: text,
		})
	}
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", ContentTypeText:
			if value := strings.TrimSpace(part.Text); value != "" {
				blocks = append(blocks, anthropicContentBlock{
					Type: "text",
					Text: value,
				})
			}
		case ContentTypeImage:
			if part.Image == nil {
				continue
			}
			source, err := toAnthropicImageSource(part.Image)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, anthropicContentBlock{
				Type:   "image",
				Source: source,
			})
		}
	}
	if len(blocks) == 0 {
		return msg.Text, nil
	}
	if len(blocks) == 1 && blocks[0].Type == "text" {
		return blocks[0].Text, nil
	}
	return blocks, nil
}

func toAnthropicImageSource(image *ImageContent) (*anthropicImageSource, error) {
	source, err := resolveImageSource(image)
	if err != nil {
		return nil, err
	}
	if source.URL != "" {
		return &anthropicImageSource{
			Type: "url",
			URL:  source.URL,
		}, nil
	}
	return &anthropicImageSource{
		Type:      "base64",
		MediaType: source.MediaType,
		Data:      source.Base64Data,
	}, nil
}
