// Ask-human tool implementation that pauses the agent and resumes after user input.

package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

type AskHumanTool struct{}

type askHumanArgs struct {
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

type askHumanAwaitingPayload struct {
	Status        string           `json:"status"`
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

// NewAskHumanTool 创建 ask_human 工具实例。
func NewAskHumanTool() Tool {
	return AskHumanTool{}
}

func (AskHumanTool) Name() string {
	return "ask_human"
}

func (AskHumanTool) Description() string {
	return "Pause and ask the user for required input before continuing, optionally with single-choice or multi-choice options."
}

func (AskHumanTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"prompt":{"type":"string","description":"Question for the user."},
			"selection_mode":{"type":"string","enum":["single","multiple"],"description":"Optional selection mode when presenting predefined choices."},
			"options":{
				"type":"array",
				"description":"Optional selectable choices. If provided, the final option must set allow_custom=true so the user can type a custom answer.",
				"items":{
					"type":"object",
					"properties":{
						"label":{"type":"string","description":"User-visible option label."},
						"allow_custom":{"type":"boolean","description":"Only set true on the final option to allow free-text input."}
					},
					"required":["label"],
					"additionalProperties":false
				}
			}
		},
		"required":["prompt"],
		"additionalProperties":false
	}`)
}

// InterpretResult 从工具输出中提取暂停信号，交给 Agent 上抛等待态。
func (AskHumanTool) InterpretResult(output string) ExecuteMeta {
	var payload askHumanAwaitingPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ExecuteMeta{}
	}
	if payload.Status != "awaiting_human" {
		return ExecuteMeta{}
	}

	questionID := strings.TrimSpace(payload.QuestionID)
	prompt := strings.TrimSpace(payload.Prompt)
	if questionID == "" || prompt == "" {
		return ExecuteMeta{}
	}

	options, ok := normalizeAskHumanOptions(payload.Options)
	if !ok {
		return ExecuteMeta{}
	}
	selectionMode, ok := normalizeAskHumanSelectionMode(payload.SelectionMode, len(options) > 0)
	if !ok {
		return ExecuteMeta{}
	}

	return ExecuteMeta{
		AwaitingHuman: &AwaitingHumanSignal{
			QuestionID:    questionID,
			Prompt:        prompt,
			SelectionMode: selectionMode,
			Options:       options,
		},
	}
}

// Execute 校验 prompt 后，把“等待人工回答”的状态写入当前会话。
func (AskHumanTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	var args askHumanArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	question, err := normalizeAskHumanQuestion(args)
	if err != nil {
		return "", err
	}
	return askHumanExecute(ctx, question, traceID)
}

// askHumanExecute 依赖会话上下文与 tool_call_id，注册 pending question 并返回等待态 payload。
func askHumanExecute(ctx context.Context, question askHumanArgs, traceID string) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("ask_human requires an active session")
	}

	toolCallID := ToolCallIDFromContext(ctx)
	if toolCallID == "" {
		return "", fmt.Errorf("ask_human requires tool call id in context")
	}

	questionID, err := newQuestionID()
	if err != nil {
		return "", fmt.Errorf("generate question id: %w", err)
	}

	sess.AddPendingQuestion(questionID, session.PendingHumanQuestion{
		Prompt:        question.Prompt,
		SelectionMode: question.SelectionMode,
		Options:       sessionOptionsFromAskHuman(question.Options),
		ToolCallID:    toolCallID,
		TraceID:       strings.TrimSpace(traceID),
		CreatedAt:     time.Now().UTC(),
	})

	payload := askHumanAwaitingPayload{
		Status:        "awaiting_human",
		QuestionID:    questionID,
		Prompt:        question.Prompt,
		SelectionMode: question.SelectionMode,
		Options:       cloneAskHumanOptions(question.Options),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode awaiting payload: %w", err)
	}
	return string(encoded), nil
}

// newQuestionID 生成全局低冲突问题 ID，用于后续 HUMAN_RESPONSE 关联。
func newQuestionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "q-" + hex.EncodeToString(raw[:]), nil
}

func normalizeAskHumanQuestion(args askHumanArgs) (askHumanArgs, error) {
	prompt := strings.TrimSpace(args.Prompt)
	if prompt == "" {
		return askHumanArgs{}, fmt.Errorf("prompt is required")
	}

	options, ok := normalizeAskHumanOptions(args.Options)
	if !ok {
		return askHumanArgs{}, fmt.Errorf("options must have non-empty labels and only the final option may allow custom input")
	}
	selectionMode, ok := normalizeAskHumanSelectionMode(args.SelectionMode, len(options) > 0)
	if !ok {
		return askHumanArgs{}, fmt.Errorf("selection_mode must be empty, %q, or %q", session.HumanQuestionSelectionSingle, session.HumanQuestionSelectionMultiple)
	}
	if len(options) > 0 && len(options) < 2 {
		return askHumanArgs{}, fmt.Errorf("options must include at least one predefined choice and a final custom option")
	}
	if len(options) > 0 && !options[len(options)-1].AllowCustom {
		return askHumanArgs{}, fmt.Errorf("the final option must allow custom input")
	}

	return askHumanArgs{
		Prompt:        prompt,
		SelectionMode: selectionMode,
		Options:       options,
	}, nil
}

func normalizeAskHumanSelectionMode(raw string, hasOptions bool) (string, bool) {
	selectionMode := strings.ToLower(strings.TrimSpace(raw))
	if !hasOptions {
		return "", selectionMode == ""
	}
	if selectionMode == "" {
		return session.HumanQuestionSelectionSingle, true
	}
	switch selectionMode {
	case session.HumanQuestionSelectionSingle, session.HumanQuestionSelectionMultiple:
		return selectionMode, true
	default:
		return "", false
	}
}

func normalizeAskHumanOptions(options []AskHumanOption) ([]AskHumanOption, bool) {
	if len(options) == 0 {
		return nil, true
	}
	normalized := make([]AskHumanOption, 0, len(options))
	for index, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			return nil, false
		}
		if option.AllowCustom && index != len(options)-1 {
			return nil, false
		}
		normalized = append(normalized, AskHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	if len(normalized) == 0 {
		return nil, true
	}
	return normalized, true
}

func cloneAskHumanOptions(options []AskHumanOption) []AskHumanOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]AskHumanOption, 0, len(options))
	for _, option := range options {
		cloned = append(cloned, AskHumanOption{
			Label:       option.Label,
			AllowCustom: option.AllowCustom,
		})
	}
	return cloned
}

func sessionOptionsFromAskHuman(options []AskHumanOption) []session.HumanQuestionOption {
	if len(options) == 0 {
		return nil
	}
	mapped := make([]session.HumanQuestionOption, 0, len(options))
	for _, option := range options {
		mapped = append(mapped, session.HumanQuestionOption{
			Label:       option.Label,
			AllowCustom: option.AllowCustom,
		})
	}
	return mapped
}
