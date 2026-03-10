package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

// SessionHistoryBuilder 负责会话装载与 Agent 历史恢复。
type SessionHistoryBuilder struct {
	provider     ProviderConfig
	systemPrompt string
	sessionStore *session.Store
}

func newSessionHistoryBuilder(provider ProviderConfig, systemPrompt string, sessionStore *session.Store) *SessionHistoryBuilder {
	return &SessionHistoryBuilder{
		provider:     provider,
		systemPrompt: strings.TrimSpace(systemPrompt),
		sessionStore: sessionStore,
	}
}

// LoadOrCreateSession 在 session_id 为空（或未启用持久化存储）时创建新会话；否则仅加载已存在会话。
func (b *SessionHistoryBuilder) LoadOrCreateSession(sessionID string) (*session.Session, error) {
	trimmedSessionID := strings.TrimSpace(sessionID)
	prompt := ""
	if b != nil {
		prompt = b.systemPrompt
	}
	if trimmedSessionID == "" || b == nil || b.sessionStore == nil {
		// 首次会话没有历史，按当前 system prompt 创建空会话。
		return session.NewSession(prompt), nil
	}

	sess, err := b.sessionStore.Load(trimmedSessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, fmt.Errorf("%w: session_id=%s", session.ErrSessionNotFound, trimmedSessionID)
		}
		return nil, err
	}
	return sess, nil
}

// BuildHistory 基于会话历史恢复 Agent 上下文，并注入已回答的人类反馈。
func (b *SessionHistoryBuilder) BuildHistory(sess *session.Session) *agent.History {
	history, _ := b.BuildHistoryWithResolvedQuestions(sess)
	return history
}

// BuildHistoryWithResolvedQuestions 返回恢复后的历史，以及本轮前刚被消费的人类回答。
func (b *SessionHistoryBuilder) BuildHistoryWithResolvedQuestions(sess *session.Session) (*agent.History, []memory.DecisionAnsweredQuestion) {
	// 把上一轮已回答的 ask_human 结果注入为 tool 消息，恢复中断链路。
	resolved := injectAnsweredHumanResponses(sess)

	if b == nil {
		return agent.NewHistoryFromMessages(nil), resolved
	}

	contextLimit := session.GetContextLimit(b.provider.Type, b.provider.Model, session.ContextLimitConfig{
		ContextWindowTokens:        b.provider.ContextWindowTokens,
		ResponseReserveTokens:      b.provider.ResponseReserveTokens,
		ModelContextWindowTokens:   b.provider.ModelContextWindowTokens,
		ModelResponseReserveTokens: b.provider.ModelResponseReserveTokens,
	})
	messages := messagesWithSystemPrompt(sess.GetMessages(contextLimit), b.systemPrompt)
	history := agent.NewHistoryFromMessages(messages)
	if sess != nil && !sess.ConversationState.IsZero() && sess.ConversationState.Matches(b.provider.Type, b.provider.BaseURL, b.provider.Model) {
		history.SetConversationState(sess.ConversationState)
	}
	return history, resolved
}

// injectAnsweredHumanResponses 将会话中已答复的人类问题转换为 tool result 消息。
func injectAnsweredHumanResponses(sess *session.Session) []memory.DecisionAnsweredQuestion {
	if sess == nil {
		return nil
	}

	// PopAnsweredQuestions 只返回“尚未注入历史”的回答，避免重复消费。
	resolved := sess.PopAnsweredQuestions()
	if len(resolved) == 0 {
		return nil
	}

	out := make([]memory.DecisionAnsweredQuestion, 0, len(resolved))
	answeredAt := time.Now().UTC()
	for _, item := range resolved {
		out = append(out, memory.DecisionAnsweredQuestion{
			QuestionID: item.QuestionID,
			Prompt:     item.Question.Prompt,
			ToolCallID: item.Question.ToolCallID,
			TraceID:    item.Question.TraceID,
			Answer:     item.Answer,
			AskedAt:    item.Question.CreatedAt,
			AnsweredAt: answeredAt,
		})

		toolCallID := strings.TrimSpace(item.Question.ToolCallID)
		if toolCallID == "" {
			continue
		}

		payload := map[string]any{
			"question_id": item.QuestionID,
			"prompt":      item.Question.Prompt,
			"answer":      item.Answer,
		}
		if selectionMode := strings.TrimSpace(item.Question.SelectionMode); selectionMode != "" {
			payload["selection_mode"] = selectionMode
		}
		if len(item.Question.Options) > 0 {
			options := make([]map[string]any, 0, len(item.Question.Options))
			for _, option := range item.Question.Options {
				label := strings.TrimSpace(option.Label)
				if label == "" {
					continue
				}
				options = append(options, map[string]any{
					"label":        label,
					"allow_custom": option.AllowCustom,
				})
			}
			if len(options) > 0 {
				payload["options"] = options
			}
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			encoded = []byte(`{}`)
		}

		traceID := strings.TrimSpace(item.Question.TraceID)
		// 对齐 agent 侧 tool result envelope，保证后续轮次可无缝推理。
		sess.AddMessage(agentMessageForAskHuman(toolCallID, traceID, string(encoded)))
	}
	return out
}

// agentMessageForAskHuman 构造 ask_human 的 tool 消息，供下一轮继续推理。
func agentMessageForAskHuman(toolCallID string, traceID string, output string) llm.Message {
	return llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       agent.FormatToolResult("ask_human", traceID, output, nil),
	}
}

// messagesWithSystemPrompt 在请求构建阶段覆盖 system prompt，避免修改会话持久化内容。
func messagesWithSystemPrompt(messages []llm.Message, systemPrompt string) []llm.Message {
	out := llm.CloneMessages(messages)
	prompt := strings.TrimSpace(systemPrompt)
	if prompt == "" {
		return out
	}

	desired := llm.Message{
		Role: llm.RoleSystem,
		Text: prompt,
	}

	switch {
	case len(out) == 0:
		return []llm.Message{desired}
	case out[0].Role == llm.RoleSystem:
		out[0] = desired
		return out
	default:
		updated := make([]llm.Message, 0, len(out)+1)
		updated = append(updated, desired)
		updated = append(updated, out...)
		return updated
	}
}
