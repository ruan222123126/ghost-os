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
	Prompt string `json:"prompt"`
}

type askHumanAwaitingPayload struct {
	Status     string `json:"status"`
	QuestionID string `json:"question_id"`
	Prompt     string `json:"prompt"`
}

// NewAskHumanTool 创建 ask_human 工具实例。
func NewAskHumanTool() Tool {
	return AskHumanTool{}
}

func (AskHumanTool) Name() string {
	return "ask_human"
}

func (AskHumanTool) Description() string {
	return "Pause autonomous execution and ask the user for explicit input before continuing."
}

func (AskHumanTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"prompt":{"type":"string","description":"Question that should be presented to the user."}
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

	return ExecuteMeta{
		AwaitingHuman: &AwaitingHumanSignal{
			QuestionID: questionID,
			Prompt:     prompt,
		},
	}
}

// Execute 校验 prompt 后，把“等待人工回答”的状态写入当前会话。
func (AskHumanTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	var args askHumanArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	prompt := strings.TrimSpace(args.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	return askHumanExecute(ctx, prompt, traceID)
}

// askHumanExecute 依赖会话上下文与 tool_call_id，注册 pending question 并返回等待态 payload。
func askHumanExecute(ctx context.Context, prompt string, traceID string) (string, error) {
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
		Prompt:     prompt,
		ToolCallID: toolCallID,
		TraceID:    strings.TrimSpace(traceID),
		CreatedAt:  time.Now().UTC(),
	})

	payload := askHumanAwaitingPayload{
		Status:     "awaiting_human",
		QuestionID: questionID,
		Prompt:     prompt,
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
