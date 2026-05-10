package orchestration

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

const (
	sessionTitleAction    = "SESSION_TITLE_GENERATE"
	sessionTitleRuneLimit = 48
	titleEllipsis         = "..."
	titleEllipsisRunes    = len(titleEllipsis)
)

type sessionTitleTask struct {
	store     *session.Store
	completer agent.Completer
	sessionID string
	message   string
	traceID   string
}

func (p *sessionTurnPreparer) prepareCreatedSessionTitleTask(
	sess *session.Session,
	deps agentRuntimeDependencies,
	input turnPreparationInput,
) *sessionTitleTask {
	if sess == nil || strings.TrimSpace(input.userMessage) == "" {
		return nil
	}
	switch strings.TrimSpace(deps.cfg.SessionTitleMode) {
	case bridgeconfig.SessionTitleModeFirstMessage:
		sess.Title = titleFromFirstMessage(input.userMessage)
	case bridgeconfig.SessionTitleModeAIGenerated:
		return p.newAITitleTask(sess.ID, deps.client, input)
	}
	return nil
}

func (p *sessionTurnPreparer) newAITitleTask(
	sessionID string,
	completer agent.Completer,
	input turnPreparationInput,
) *sessionTitleTask {
	if p == nil || p.sessionStore == nil {
		return nil
	}
	return &sessionTitleTask{
		store:     p.sessionStore,
		completer: completer,
		sessionID: strings.TrimSpace(sessionID),
		message:   strings.TrimSpace(input.userMessage),
		traceID:   strings.TrimSpace(input.traceID),
	}
}

func startSessionTitleTask(task *sessionTitleTask) {
	if task == nil {
		return
	}
	go task.run()
}

func (t *sessionTitleTask) run() {
	if t.store == nil || t.completer == nil {
		logAction(t.traceID, sessionTitleAction, "error", errors.New("session title generator is not configured"))
		return
	}
	title, err := generateAITitle(context.Background(), t.completer, t.message)
	if err != nil {
		logAction(t.traceID, sessionTitleAction, "error", err)
		return
	}
	if err := t.store.UpdateTitle(t.sessionID, title); err != nil {
		logAction(t.traceID, sessionTitleAction, "error", err)
	}
}

func generateAITitle(ctx context.Context, completer agent.Completer, message string) (string, error) {
	response, err := completer.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: sessionTitlePrompt()},
			{Role: llm.RoleUser, Text: strings.TrimSpace(message)},
		},
	})
	if err != nil {
		return "", err
	}
	if response == nil {
		return "", errors.New("session title generator returned nil response")
	}
	title := compactWhitespace(response.Message.Text)
	if title == "" {
		return "", errors.New("session title generator returned empty title")
	}
	return title, nil
}

func sessionTitlePrompt() string {
	return strings.Join([]string{
		"你是会话标题生成器。",
		"基于用户首条消息生成一个简短标题，最多 48 个字符。",
		"只返回标题本身，不要 Markdown、不要解释、不要引号。",
	}, "\n")
}

func titleFromFirstMessage(message string) string {
	segment := firstNonEmptySegment(message)
	if segment == "" {
		return ""
	}
	return truncateTitle(compactWhitespace(segment), sessionTitleRuneLimit)
}

func firstNonEmptySegment(message string) string {
	for _, line := range strings.Split(message, "\n") {
		segment := firstSentence(strings.TrimSpace(line))
		if segment != "" {
			return segment
		}
	}
	return ""
}

func firstSentence(segment string) string {
	for index, char := range segment {
		if isSentenceTerminator(char) {
			return strings.TrimSpace(segment[:index+len(string(char))])
		}
	}
	return strings.TrimSpace(segment)
}

func isSentenceTerminator(char rune) bool {
	switch char {
	case '.', '?', '!', '。', '？', '！':
		return true
	default:
		return false
	}
}

func compactWhitespace(value string) string {
	return strings.Join(strings.FieldsFunc(value, unicode.IsSpace), " ")
}

func truncateTitle(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= titleEllipsisRunes {
		return string(runes[:limit])
	}
	return string(runes[:limit-titleEllipsisRunes]) + titleEllipsis
}
