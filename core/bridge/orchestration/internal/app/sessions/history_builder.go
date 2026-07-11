package sessions

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
)

// HistoryBuilder 负责会话装载与 Agent 历史恢复。
type HistoryBuilder struct {
	provider            sessionturn.ProviderContext
	systemPrompt        string
	sessionStore        *session.Store
	idleTurns           int
	microcompactEnabled bool
	traceID             string
}

func NewHistoryBuilder(
	provider sessionturn.ProviderContext,
	systemPrompt string,
	sessionStore *session.Store,
	idleTurns int,
	microcompactEnabled bool,
	traceID string,
) *HistoryBuilder {
	return &HistoryBuilder{
		provider:            provider,
		systemPrompt:        strings.TrimSpace(systemPrompt),
		sessionStore:        sessionStore,
		idleTurns:           idleTurns,
		microcompactEnabled: microcompactEnabled,
		traceID:             strings.TrimSpace(traceID),
	}
}

func (b *HistoryBuilder) SetTraceID(traceID string) {
	if b != nil {
		b.traceID = strings.TrimSpace(traceID)
	}
}

// LoadOrCreateSession 在 session_id 为空（或未启用持久化存储）时创建新会话；否则仅加载已存在会话。
// 返回值 created=true 表示本次返回的是新建会话。
func (b *HistoryBuilder) LoadOrCreateSession(sessionID string) (*session.Session, bool, error) {
	trimmedSessionID := strings.TrimSpace(sessionID)
	prompt := ""
	if b != nil {
		prompt = b.systemPrompt
	}
	if trimmedSessionID == "" || b == nil || b.sessionStore == nil {
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
func (b *HistoryBuilder) BuildHistory(sess *session.Session) *agent.History {
	history, _ := b.BuildHistoryWithResolvedQuestions(sess)
	return history
}

// BuildHistoryWithResolvedQuestions 返回恢复后的历史，以及本轮前刚被消费的人类回答。
func (b *HistoryBuilder) BuildHistoryWithResolvedQuestions(
	sess *session.Session,
) (*agent.History, []sessionturn.ResolvedHumanQuestion) {
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
	rawMessages := []llm.Message(nil)
	if sess != nil {
		rawMessages = sess.Messages
	}
	projected := sessionturn.ProjectMessagesForModel(rawMessages, sessionturn.ProjectionOptions{
		IdleTurns:           b.idleTurns,
		MicrocompactEnabled: b.microcompactEnabled,
		TraceID:             b.traceID,
	})
	sanitized := sessionturn.SanitizeToolProtocolMessages(projected)
	pruned := session.PruneMessages(sanitized, contextLimit)
	messages := sessionturn.MessagesWithSystemPrompt(pruned, b.systemPrompt)
	history := agent.NewHistoryFromMessages(messages)
	if sess != nil && !sess.ConversationState.IsZero() && sess.ConversationState.Matches(b.provider.Type, b.provider.BaseURL, b.provider.Model) {
		history.SetConversationState(sess.ConversationState)
	}
	return history, resolved
}

func injectAnsweredHumanResponses(sess *session.Session) []sessionturn.ResolvedHumanQuestion {
	if sess == nil {
		return nil
	}

	resolved := sess.PopAnsweredQuestions()
	if len(resolved) == 0 {
		return nil
	}

	answeredAt := time.Now().UTC()
	out := make([]sessionturn.ResolvedHumanQuestion, 0, len(resolved))
	for _, item := range resolved {
		out = append(out, sessionturn.ResolvedHumanQuestion{
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
		toolName, output, summary := sessionturn.BuildResolvedHumanQuestionToolResult(answeredHumanQuestionInput(item))
		out[len(out)-1].ToolName = toolName
		out[len(out)-1].Summary = summary
		traceID := strings.TrimSpace(item.Question.TraceID)
		sess.AddMessage(sessionturn.AgentMessageForResolvedHumanTool(toolCallID, toolName, traceID, output))
	}
	return out
}

func answeredHumanQuestionInput(item session.AnsweredHumanQuestion) sessionturn.AnsweredHumanQuestionInput {
	return sessionturn.AnsweredHumanQuestionInput{
		QuestionID:    item.QuestionID,
		Prompt:        item.Question.Prompt,
		SelectionMode: item.Question.SelectionMode,
		Options:       humanQuestionOptionInputs(item.Question.Options),
		Answer:        item.Answer,
	}
}

func humanQuestionOptionInputs(raw []session.HumanQuestionOption) []sessionturn.HumanQuestionOptionInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.HumanQuestionOptionInput, 0, len(raw))
	for _, option := range raw {
		out = append(out, sessionturn.HumanQuestionOptionInput{
			Label:       option.Label,
			AllowCustom: option.AllowCustom,
		})
	}
	return out
}
