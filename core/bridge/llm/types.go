package llm

import (
	"encoding/json"
	"strings"
)

// Role 是内部统一的消息角色集合。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// FinishReason 是内部统一的回合结束原因。
type FinishReason string

const (
	FinishStop      FinishReason = "stop"
	FinishToolCalls FinishReason = "tool_calls"
	FinishLength    FinishReason = "length"
)

// Message 是 provider 无关的会话消息结构。
type Message struct {
	Role       Role
	Text       string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolCall 描述模型发起的一次工具调用。
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// ToolDef 是暴露给模型的工具定义。
type ToolDef struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// CompletionRequest 是一次模型请求的统一输入。
type CompletionRequest struct {
	Messages []Message
	Tools    []ToolDef
}

// CompletionResponse 是一次模型请求的统一输出。
type CompletionResponse struct {
	Message      Message
	FinishReason FinishReason
	Usage        Usage
}

// Usage 是统一 token 统计结构。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// CloneMessages 对消息做深拷贝，避免跨层共享可变切片。
func CloneMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]Message, len(messages))
	for i, msg := range messages {
		out[i] = Message{
			Role:       msg.Role,
			Text:       msg.Text,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ToolCalls) > 0 {
			out[i].ToolCalls = cloneToolCalls(msg.ToolCalls)
		}
	}

	return out
}

func cloneToolCalls(calls []ToolCall) []ToolCall {
	out := make([]ToolCall, len(calls))
	for i, call := range calls {
		out[i] = ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: cloneRawJSON(call.Arguments),
		}
	}

	return out
}

// cloneRawJSON 复制原始 JSON 字节，保证调用方可安全持有。
func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

// normalizeJSONObject 把空值归一化为 {}，减少 provider 分支判断。
func normalizeJSONObject(raw json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return json.RawMessage(`{}`)
	}
	return cloneRawJSON(raw)
}

// contentToText 把 provider content 折叠为文本，兜底为 JSON 字符串。
func contentToText(content any) string {
	if content == nil {
		return ""
	}

	if text, ok := content.(string); ok {
		return text
	}

	encoded, err := json.Marshal(content)
	if err != nil {
		return ""
	}
	return string(encoded)
}
