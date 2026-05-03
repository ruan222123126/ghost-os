package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var errToolCallIDEmpty = errors.New("tool_call.id is empty")
var errToolCallNameEmpty = errors.New("tool_call.name is empty")
var errToolCallArgumentsEmpty = errors.New("tool_call.arguments is empty")
var errToolCallArgumentsObject = errors.New("tool_call.arguments must be a JSON object")
var errToolResultCallIDEmpty = errors.New("tool message.tool_call_id is empty")
var errAssistantReasoningReplayMissing = errors.New("assistant tool_calls replay is missing reasoning_content after thinking mode started")

func validateRequestMessageToolProtocol(messages []Message) error {
	pending := make(map[string]struct{})

	for messageIndex, msg := range messages {
		switch msg.Role {
		case RoleAssistant:
			for toolIndex, call := range msg.ToolCalls {
				toolCallID := strings.TrimSpace(call.ID)
				if toolCallID == "" {
					return fmt.Errorf("messages[%d].tool_calls[%d]: %w", messageIndex, toolIndex, errToolCallIDEmpty)
				}
				if strings.TrimSpace(call.Name) == "" {
					return fmt.Errorf("messages[%d].tool_calls[%d]: %w", messageIndex, toolIndex, errToolCallNameEmpty)
				}
				if err := validateToolCallArgumentsObject(call.Arguments); err != nil {
					return fmt.Errorf("messages[%d].tool_calls[%d]: %w", messageIndex, toolIndex, err)
				}
				if _, exists := pending[toolCallID]; exists {
					return fmt.Errorf("messages[%d].tool_calls[%d]: duplicate unmatched tool_call.id %q", messageIndex, toolIndex, toolCallID)
				}
				pending[toolCallID] = struct{}{}
			}
		case RoleTool:
			toolCallID := strings.TrimSpace(msg.ToolCallID)
			if toolCallID == "" {
				return fmt.Errorf("messages[%d]: %w", messageIndex, errToolResultCallIDEmpty)
			}
			if _, ok := pending[toolCallID]; !ok {
				return fmt.Errorf("messages[%d]: tool message references unknown tool_call_id %q", messageIndex, toolCallID)
			}
			delete(pending, toolCallID)
		}
	}

	if len(pending) == 0 {
		return ValidateReasoningReplay(messages)
	}

	ids := make([]string, 0, len(pending))
	for id := range pending {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return fmt.Errorf("assistant tool_calls are missing matching tool results: %s", strings.Join(ids, ", "))
}

// ValidateReasoningReplay ensures a single user-turn tool loop does not drop
// reasoning_content on later assistant messages once a prior assistant tool
// turn in that loop has entered thinking mode.
func ValidateReasoningReplay(messages []Message) error {
	segmentStart := 0
	for index, msg := range messages {
		if msg.Role != RoleUser {
			continue
		}
		if err := validateReasoningReplaySegment(messages[segmentStart:index], segmentStart); err != nil {
			return err
		}
		segmentStart = index + 1
	}
	return validateReasoningReplaySegment(messages[segmentStart:], segmentStart)
}

func validateReasoningReplaySegment(messages []Message, offset int) error {
	thinkingStarted := false
	for index, msg := range messages {
		if msg.Role != RoleAssistant {
			continue
		}
		if hasReasoningReplayContent(msg.ReasoningContent) {
			if len(msg.ToolCalls) > 0 || thinkingStarted {
				thinkingStarted = true
			}
			continue
		}
		if thinkingStarted {
			return fmt.Errorf("messages[%d]: %w", offset+index, errAssistantReasoningReplayMissing)
		}
	}
	return nil
}

func hasReasoningReplayContent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func validateToolCallArgumentsObject(raw json.RawMessage) error {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return errToolCallArgumentsEmpty
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("%w: %v", errToolCallArgumentsObject, err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return errToolCallArgumentsObject
	}
	return nil
}
