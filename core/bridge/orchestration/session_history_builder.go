package orchestration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

// SessionHistoryBuilder 负责会话装载与 Agent 历史恢复。
type SessionHistoryBuilder struct {
	provider     bridgeconfig.ProviderConfig
	systemPrompt string
	sessionStore *session.Store
	idleTurns    int
}

type resolvedHumanQuestion struct {
	QuestionID string
	ToolName   string
	Prompt     string
	ToolCallID string
	TraceID    string
	Answer     string
	Summary    string
	AskedAt    time.Time
	AnsweredAt time.Time
}

func newSessionHistoryBuilder(
	provider bridgeconfig.ProviderConfig,
	systemPrompt string,
	sessionStore *session.Store,
	idleTurns int,
) *SessionHistoryBuilder {
	return &SessionHistoryBuilder{
		provider:     provider,
		systemPrompt: strings.TrimSpace(systemPrompt),
		sessionStore: sessionStore,
		idleTurns:    idleTurns,
	}
}

// LoadOrCreateSession 在 session_id 为空（或未启用持久化存储）时创建新会话；否则仅加载已存在会话。
// 返回值 created=true 表示本次返回的是新建会话。
func (b *SessionHistoryBuilder) LoadOrCreateSession(sessionID string) (*session.Session, bool, error) {
	trimmedSessionID := strings.TrimSpace(sessionID)
	prompt := ""
	if b != nil {
		prompt = b.systemPrompt
	}
	if trimmedSessionID == "" || b == nil || b.sessionStore == nil {
		// 首次会话没有历史，按当前 system prompt 创建空会话。
		return session.NewSession(prompt), true, nil
	}

	sess, err := b.sessionStore.Load(trimmedSessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, false, fmt.Errorf("%w: session_id=%s", session.ErrSessionNotFound, trimmedSessionID)
		}
		return nil, false, err
	}
	return sess, false, nil
}

// BuildHistory 基于会话历史恢复 Agent 上下文，并注入已回答的人类反馈。
func (b *SessionHistoryBuilder) BuildHistory(sess *session.Session) *agent.History {
	history, _ := b.BuildHistoryWithResolvedQuestions(sess)
	return history
}

// BuildHistoryWithResolvedQuestions 返回恢复后的历史，以及本轮前刚被消费的人类回答。
func (b *SessionHistoryBuilder) BuildHistoryWithResolvedQuestions(sess *session.Session) (*agent.History, []resolvedHumanQuestion) {
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
	rawMessages := sess.GetMessages(contextLimit)
	projected := projectMessagesForModel(rawMessages, b.idleTurns)
	messages := messagesWithSystemPrompt(projected, b.systemPrompt)
	history := agent.NewHistoryFromMessages(messages)
	if sess != nil && !sess.ConversationState.IsZero() && sess.ConversationState.Matches(b.provider.Type, b.provider.BaseURL, b.provider.Model) {
		history.SetConversationState(sess.ConversationState)
	}
	return history, resolved
}

// injectAnsweredHumanResponses 将会话中已答复的人类问题转换为 tool result 消息。
func injectAnsweredHumanResponses(sess *session.Session) []resolvedHumanQuestion {
	if sess == nil {
		return nil
	}

	// PopAnsweredQuestions 只返回“尚未注入历史”的回答，避免重复消费。
	resolved := sess.PopAnsweredQuestions()
	if len(resolved) == 0 {
		return nil
	}

	answeredAt := time.Now().UTC()
	out := make([]resolvedHumanQuestion, 0, len(resolved))
	for _, item := range resolved {
		out = append(out, resolvedHumanQuestion{
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
		toolName, output, summary := resolvedHumanQuestionToolResult(sess, item)
		out[len(out)-1].ToolName = toolName
		out[len(out)-1].Summary = summary
		traceID := strings.TrimSpace(item.Question.TraceID)
		sess.AddMessage(agentMessageForResolvedHumanTool(toolCallID, toolName, traceID, output))
	}
	return out
}

func agentMessageForResolvedHumanTool(
	toolCallID string,
	toolName string,
	traceID string,
	output string,
) llm.Message {
	return llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       agent.FormatToolResult(toolName, traceID, output, nil),
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
