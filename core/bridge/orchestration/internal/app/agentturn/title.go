package agentturn

import (
	"context"
	"errors"
	"strings"
	"unicode"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
)

const (
	SessionTitleAction    = "SESSION_TITLE_GENERATE"
	SessionTitleRuneLimit = 48
	titleEllipsis         = "..."
	titleEllipsisRunes    = len(titleEllipsis)
)

type SessionTitleStore interface {
	UpdateTitle(sessionID string, title string) error
}

type SessionTitlePreparationInput struct {
	Mode        string
	Store       SessionTitleStore
	Completer   llm.Completer
	Logger      Logger
	SessionID   string
	UserMessage string
	TraceID     string
}

type SessionTitlePreparation struct {
	ImmediateTitle    string
	HasImmediateTitle bool
	Task              *SessionTitleTask
}

type SessionTitleTask struct {
	store     SessionTitleStore
	completer llm.Completer
	logger    Logger
	sessionID string
	message   string
	traceID   string
}

func PrepareCreatedSessionTitle(input SessionTitlePreparationInput) SessionTitlePreparation {
	if strings.TrimSpace(input.UserMessage) == "" {
		return SessionTitlePreparation{}
	}
	switch strings.TrimSpace(input.Mode) {
	case bridgeconfig.SessionTitleModeFirstMessage:
		return SessionTitlePreparation{
			ImmediateTitle:    TitleFromFirstMessage(input.UserMessage),
			HasImmediateTitle: true,
		}
	case bridgeconfig.SessionTitleModeAIGenerated:
		return SessionTitlePreparation{Task: NewAITitleTask(input)}
	default:
		return SessionTitlePreparation{}
	}
}

func NewAITitleTask(input SessionTitlePreparationInput) *SessionTitleTask {
	if input.Store == nil {
		return nil
	}
	return &SessionTitleTask{
		store:     input.Store,
		completer: input.Completer,
		logger:    input.Logger,
		sessionID: strings.TrimSpace(input.SessionID),
		message:   strings.TrimSpace(input.UserMessage),
		traceID:   strings.TrimSpace(input.TraceID),
	}
}

func StartSessionTitleTask(task *SessionTitleTask) {
	if task == nil {
		return
	}
	go task.run()
}

func (t *SessionTitleTask) run() {
	if t.store == nil || t.completer == nil {
		t.log("error", errors.New("session title generator is not configured"))
		return
	}
	title, err := generateAITitle(context.Background(), t.completer, t.message)
	if err != nil {
		t.log("error", err)
		return
	}
	if err := t.store.UpdateTitle(t.sessionID, title); err != nil {
		t.log("error", err)
	}
}

func (t *SessionTitleTask) log(status string, err error) {
	if t.logger == nil {
		return
	}
	t.logger.Log(t.traceID, SessionTitleAction, status, err)
}

func generateAITitle(ctx context.Context, completer llm.Completer, message string) (string, error) {
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

func TitleFromFirstMessage(message string) string {
	segment := firstNonEmptySegment(message)
	if segment == "" {
		return ""
	}
	return truncateTitle(compactWhitespace(segment), SessionTitleRuneLimit)
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
