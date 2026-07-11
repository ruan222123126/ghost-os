package agent

import "ghost-os/bridge/llm"

func validateAssistantReasoningReplay(history *History, candidate llm.Message) error {
	if history == nil || candidate.Role != llm.RoleAssistant {
		return nil
	}

	messages := history.Messages()
	cloned := llm.CloneMessages([]llm.Message{candidate})
	if len(cloned) == 1 {
		messages = append(messages, cloned[0])
	}
	return llm.ValidateReasoningReplay(messages)
}
