package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	orchestrationDispatchToolName = "orchestration_dispatch"
	dispatchOrderSequential       = "sequential"
)

func (b *runTranscriptBuilder) addOrchestrationDispatchToolMessage(
	groupTitle string,
	groupID string,
	dispatch map[string]any,
) {
	b.addToolMessage(runToolMessage{
		sender:    orchestrationDispatchSender(groupTitle, groupID, transcriptInt(dispatch, "round")),
		toolName:  orchestrationDispatchToolName,
		arguments: orchestrationDispatchToolArguments(dispatch),
		output:    orchestrationDispatchToolOutput(dispatch),
		status:    runStatusSuccess,
	})
}

func orchestrationDispatchSender(groupTitle string, groupID string, round int) string {
	label := strings.TrimSpace(groupTitle)
	if label != "" && strings.TrimSpace(groupID) != "" && label != strings.TrimSpace(groupID) {
		label += " / " + strings.TrimSpace(groupID)
	}
	if label == "" {
		label = strings.TrimSpace(groupID)
	}
	if round > 0 {
		return fmt.Sprintf("群主调度（%s） · 第 %d 轮", label, round)
	}
	return fmt.Sprintf("群主调度（%s）", label)
}

func orchestrationDispatchToolArguments(dispatch map[string]any) map[string]any {
	action := transcriptString(dispatch, "action")
	args := map[string]any{"action": action}
	if action == "private_send" {
		if messages := orchestrationDispatchPrivateMessages(dispatch); len(messages) > 0 {
			args["private_messages"] = messages
		}
		return args
	}
	if participants := transcriptStringSlice(dispatch["participant_ids"]); len(participants) > 0 {
		args["participant_ids"] = participants
	}
	if order := transcriptString(dispatch, "order"); order != "" && order != dispatchOrderSequential {
		args["order"] = order
	}
	if instruction := transcriptString(dispatch, "instruction"); instruction != "" {
		args["instruction"] = instruction
	}
	return args
}

func orchestrationDispatchPrivateMessages(dispatch map[string]any) []map[string]string {
	deliveries := transcriptSlice(dispatch, "private_deliveries")
	if len(deliveries) == 0 {
		return nil
	}
	messages := make([]map[string]string, 0, len(deliveries))
	for _, item := range deliveries {
		delivery := transcriptRecord(item)
		participantID := transcriptString(delivery, "participant_id")
		if participantID == "" {
			continue
		}
		messages = append(messages, map[string]string{
			"participant_id": participantID,
			"content":        transcriptString(delivery, "content"),
		})
	}
	return messages
}

func orchestrationDispatchToolOutput(dispatch map[string]any) string {
	encoded, err := json.Marshal(dispatch)
	if err != nil {
		return ""
	}
	return string(encoded)
}
