package llm

import (
	"context"
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
	RoleInternal  Role = "internal"
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
	Role    Role
	Text    string
	Content []ContentPart
	// ReasoningContent 保留 provider 返回的 reasoning_content，供后续请求原样回传。
	ReasoningContent json.RawMessage `json:",omitempty"`
	ToolCalls        []ToolCall
	ToolCallID       string
}

const (
	ContentTypeText  = "text"
	ContentTypeImage = "image"
)

// ContentPart 表示一段可序列化的多模态消息内容。
type ContentPart struct {
	Type  string        `json:"type"`
	Text  string        `json:"text,omitempty"`
	Image *ImageContent `json:"image,omitempty"`
}

// ImageContent 使用本地路径或 URL 引用图片，并附带元数据。
type ImageContent struct {
	Path     string `json:"path,omitempty"`
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Bytes    int    `json:"bytes,omitempty"`
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
	Semantics   ToolSemantics
}

// ToolSemantics 描述工具对运行时协议有影响的最小语义。
type ToolSemantics struct {
	ReadOnly   bool
	SideEffect bool
}

// ConversationState 保存 provider 侧可续跑的会话状态。
type ConversationState struct {
	Provider           Provider
	BaseURL            string
	Model              string
	PreviousResponseID string
}

// ResponseOptions 是 Responses API 相关的可选请求扩展参数。
type ResponseOptions struct {
	PromptCacheKey       string
	PromptCacheRetention string
	SafetyIdentifier     string
	Metadata             map[string]string
	Store                *bool
}

func (o ResponseOptions) IsZero() bool {
	return strings.TrimSpace(o.PromptCacheKey) == "" &&
		strings.TrimSpace(o.PromptCacheRetention) == "" &&
		strings.TrimSpace(o.SafetyIdentifier) == "" &&
		len(o.Metadata) == 0 &&
		o.Store == nil
}

func (s ConversationState) IsZero() bool {
	return s.Provider == "" && strings.TrimSpace(s.BaseURL) == "" && strings.TrimSpace(s.Model) == "" && strings.TrimSpace(s.PreviousResponseID) == ""
}

func (s ConversationState) Matches(provider Provider, baseURL, model string) bool {
	return s.Provider.Normalized() == provider.Normalized() &&
		strings.TrimSpace(s.BaseURL) == strings.TrimSpace(baseURL) &&
		strings.TrimSpace(s.Model) == strings.TrimSpace(model)
}

// CompletionRequest 是一次模型请求的统一输入。
type CompletionRequest struct {
	Messages          []Message
	Tools             []ToolDef
	ConversationState ConversationState
	ResponseOptions   ResponseOptions
}

// CompletionResponse 是一次模型请求的统一输出。
type CompletionResponse struct {
	Message           Message
	FinishReason      FinishReason
	Usage             Usage
	ConversationState ConversationState
}

type Completer interface {
	Complete(context.Context, CompletionRequest) (*CompletionResponse, error)
}

// Usage 是统一 token 统计结构。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type DeltaKind string

const (
	DeltaKindText          DeltaKind = "text"
	DeltaKindThinking      DeltaKind = "thinking"
	DeltaKindToolCallStart DeltaKind = "tool_call_start"
	DeltaKindToolCallDelta DeltaKind = "tool_call_delta"
	DeltaKindToolCallEnd   DeltaKind = "tool_call_end"
)

type LLMDelta struct {
	Kind              DeltaKind
	Text              string
	Thinking          string
	ToolCallIndex     int
	ToolCallID        string
	ToolName          string
	ArgumentsFragment string
}

type LLMStreamSink interface {
	OnDelta(context.Context, LLMDelta) error
}

type StreamingCompleter interface {
	Completer
	CompleteStream(ctx context.Context, request CompletionRequest, sink LLMStreamSink) (*CompletionResponse, error)
}

// CloneMessages 对消息做深拷贝，避免跨层共享可变切片。
func CloneMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]Message, len(messages))
	for i, msg := range messages {
		out[i] = Message{
			Role:             msg.Role,
			Text:             msg.Text,
			ReasoningContent: cloneRawJSON(msg.ReasoningContent),
			ToolCallID:       msg.ToolCallID,
		}
		if len(msg.Content) > 0 {
			out[i].Content = cloneContentParts(msg.Content)
		}
		if len(msg.ToolCalls) > 0 {
			out[i].ToolCalls = cloneToolCalls(msg.ToolCalls)
		}
	}

	return out
}

func cloneContentParts(parts []ContentPart) []ContentPart {
	out := make([]ContentPart, len(parts))
	for i, part := range parts {
		out[i] = ContentPart{
			Type: part.Type,
			Text: part.Text,
		}
		if part.Image != nil {
			image := *part.Image
			out[i].Image = &image
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

func CloneResponseOptions(raw ResponseOptions) ResponseOptions {
	out := ResponseOptions{
		PromptCacheKey:       strings.TrimSpace(raw.PromptCacheKey),
		PromptCacheRetention: strings.TrimSpace(raw.PromptCacheRetention),
		SafetyIdentifier:     strings.TrimSpace(raw.SafetyIdentifier),
		Metadata:             cloneStringMap(raw.Metadata),
	}
	if raw.Store != nil {
		value := *raw.Store
		out.Store = &value
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

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
