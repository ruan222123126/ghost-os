package externalagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func approvalRequestFromRPC(id int, method string, raw json.RawMessage) (ApprovalRequest, bool) {
	params := map[string]any{}
	_ = json.Unmarshal(raw, &params)
	switch method {
	case "mcpServer/elicitation/request":
		serverName := stringValue(params["serverName"])
		callID := fmt.Sprintf("%s:%d", defaultString(serverName, "mcp"), id)
		return ApprovalRequest{
			ID:        callID,
			Kind:      "mcp",
			Tool:      defaultString(parseToolName(stringValue(params["message"])), "codex_mcp"),
			CallID:    callID,
			Prompt:    defaultString(stringValue(params["message"]), "Approve MCP tool call"),
			Payload:   params,
			MCP:       true,
			RawID:     id,
			RawMethod: method,
		}, true
	case "item/commandExecution/requestApproval", "execCommandApproval":
		callID := firstString(params["itemId"], params["callId"], fmt.Sprintf("%d", id))
		return ApprovalRequest{
			ID:        callID,
			Kind:      "exec",
			Tool:      "codex_exec",
			CallID:    callID,
			Prompt:    "Approve command execution",
			Payload:   params,
			Legacy:    method == "execCommandApproval",
			RawID:     id,
			RawMethod: method,
		}, true
	case "item/fileChange/requestApproval", "applyPatchApproval":
		callID := firstString(params["itemId"], params["callId"], fmt.Sprintf("%d", id))
		return ApprovalRequest{
			ID:        callID,
			Kind:      "patch",
			Tool:      "codex_patch",
			CallID:    callID,
			Prompt:    "Approve patch application",
			Payload:   params,
			Legacy:    method == "applyPatchApproval",
			RawID:     id,
			RawMethod: method,
		}, true
	default:
		return ApprovalRequest{}, false
	}
}

func eventFromNotification(method string, raw json.RawMessage) (CodexEvent, bool) {
	params := map[string]any{}
	_ = json.Unmarshal(raw, &params)
	if method == "codex/event" || strings.HasPrefix(method, "codex/event/") {
		msg, ok := params["msg"].(map[string]any)
		if !ok {
			return CodexEvent{}, false
		}
		eventType := stringValue(msg["type"])
		if eventType == "" && strings.HasPrefix(method, "codex/event/") {
			eventType = strings.TrimPrefix(method, "codex/event/")
			msg["type"] = eventType
		}
		return CodexEvent{Type: eventType, Method: method, Payload: msg}, eventType != ""
	}
	return rawEventFromNotification(method, params)
}

func rawEventFromNotification(method string, params map[string]any) (CodexEvent, bool) {
	switch method {
	case "turn/started":
		payload := map[string]any{"type": "task_started"}
		if turn, ok := params["turn"].(map[string]any); ok {
			payload["turn_id"] = stringValue(turn["id"])
		}
		return CodexEvent{Type: "task_started", Method: method, Payload: payload}, true
	case "turn/completed":
		payload := map[string]any{"type": "task_complete"}
		if turn, ok := params["turn"].(map[string]any); ok {
			payload["turn_id"] = stringValue(turn["id"])
			payload["status"] = turn["status"]
			payload["error"] = turn["error"]
		}
		return CodexEvent{Type: "task_complete", Method: method, Payload: payload}, true
	case "item/started", "item/completed":
		item, _ := params["item"].(map[string]any)
		return rawItemEvent(method, item)
	default:
		return CodexEvent{}, false
	}
}

func rawItemEvent(method string, item map[string]any) (CodexEvent, bool) {
	if len(item) == 0 {
		return CodexEvent{}, false
	}
	itemType := stringValue(item["type"])
	callID := stringValue(item["id"])
	switch itemType {
	case "commandExecution":
		return commandExecutionEvent(method, item, callID)
	case "fileChange":
		return fileChangeEvent(method, item, callID)
	case "agentMessage":
		return agentMessageEvent(method, item)
	default:
		return CodexEvent{}, false
	}
}

func commandExecutionEvent(method string, item map[string]any, callID string) (CodexEvent, bool) {
	if method == "item/started" {
		return CodexEvent{Type: "exec_command_begin", Method: method, Payload: map[string]any{
			"type":    "exec_command_begin",
			"call_id": callID,
			"command": item["command"],
			"cwd":     item["cwd"],
		}}, true
	}
	return CodexEvent{Type: "exec_command_end", Method: method, Payload: map[string]any{
		"type":      "exec_command_end",
		"call_id":   callID,
		"output":    item["aggregatedOutput"],
		"exit_code": item["exitCode"],
		"status":    item["status"],
	}}, true
}

func fileChangeEvent(method string, item map[string]any, callID string) (CodexEvent, bool) {
	if method == "item/started" {
		return CodexEvent{Type: "patch_apply_begin", Method: method, Payload: map[string]any{
			"type":    "patch_apply_begin",
			"call_id": callID,
			"changes": item["changes"],
		}}, true
	}
	return CodexEvent{Type: "patch_apply_end", Method: method, Payload: map[string]any{
		"type":    "patch_apply_end",
		"call_id": callID,
		"status":  item["status"],
	}}, true
}

func agentMessageEvent(method string, item map[string]any) (CodexEvent, bool) {
	if method != "item/completed" {
		return CodexEvent{}, false
	}
	text := stringValue(item["text"])
	return CodexEvent{Type: "agent_message", Method: method, Payload: map[string]any{
		"type":    "agent_message",
		"message": text,
	}}, text != ""
}

func parseToolName(message string) string {
	match := strings.Split(message, `"`)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := stringValue(value); text != "" {
			return text
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
