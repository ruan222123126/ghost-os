package agentturn

import (
	"context"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type TurnPreparationInput struct {
	StartedAt      time.Time
	RawUserMessage string
	UserMessage    string
	TraceID        string
}

type SessionRunRegistry interface {
	Register(sessionID string, traceID string, cancel context.CancelFunc) error
	Unregister(sessionID string)
}

type CompletionPromptRequest struct {
	Config               bridgeconfig.Config
	Catalog              tools.ToolCatalog
	Session              *session.Session
	SystemPrompt         string
	FallbackSystemPrompt string
	SystemPromptOverride bool
	SystemPromptFiles    *bridgeconfig.SystemPromptFiles
}

func NewTurnPreparationInput(userInput llm.Message, traceID string) TurnPreparationInput {
	message := strings.TrimSpace(userInput.Text)
	return TurnPreparationInput{
		StartedAt:      time.Now().UTC(),
		RawUserMessage: message,
		UserMessage:    message,
		TraceID:        strings.TrimSpace(traceID),
	}
}

func IsResumeLikeInput(userInput llm.Message) bool {
	return strings.TrimSpace(userInput.Text) == "" && !HasInputImages(userInput)
}

func HasAnsweredHumanResponse(sess *session.Session) bool {
	return sess != nil && len(sess.HumanAnswers) > 0
}

func RegisterSessionRun(
	ctx context.Context,
	registry SessionRunRegistry,
	sessionID string,
	traceID string,
) (context.Context, func(), error) {
	if registry == nil {
		return ctx, func() {}, nil
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := registry.Register(trimmedSessionID, strings.TrimSpace(traceID), cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		registry.Unregister(trimmedSessionID)
		cancel()
	}, nil
}

func PersistCreatedSession(sessionStore *session.Store, sess *session.Session) error {
	if sessionStore == nil || sess == nil {
		return nil
	}
	return sessionStore.Save(sess)
}

func BuildCompletionSystemPrompt(req CompletionPromptRequest) (string, error) {
	if req.SystemPromptOverride {
		return strings.TrimSpace(req.SystemPrompt), nil
	}
	if req.SystemPromptFiles != nil {
		prompt, err := bridgeruntime.BuildSystemPromptForSessionWithFiles(
			req.Config,
			req.Catalog,
			req.Session,
			req.Config.ToolSearch.IdleTurns,
			*req.SystemPromptFiles,
		)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(prompt), nil
	}
	basePrompt := strings.TrimSpace(req.SystemPrompt)
	if basePrompt == "" {
		prompt, err := bridgeruntime.BuildSystemPromptForSession(
			req.Config,
			req.Catalog,
			req.Session,
			req.Config.ToolSearch.IdleTurns,
		)
		if err != nil {
			return "", err
		}
		basePrompt = strings.TrimSpace(prompt)
	}
	if basePrompt == "" {
		basePrompt = strings.TrimSpace(req.FallbackSystemPrompt)
	}
	return basePrompt, nil
}

func AttachDynamicPromptRefresh(
	runAgent *agent.Agent,
	cfg bridgeconfig.Config,
	sess *session.Session,
	promptBuilder func() (string, error),
) {
	if runAgent == nil || sess == nil {
		return
	}
	runAgent.SetBeforeCompletionHook(func(_ context.Context, _ int, history *agent.History) error {
		if history == nil {
			return nil
		}
		PruneInvisibleSessionSkills(cfg, sess)
		if promptBuilder == nil {
			return nil
		}
		prompt, err := promptBuilder()
		if err != nil {
			return err
		}
		history.UpdateSystemPrompt(prompt)
		return nil
	})
}

func PruneInvisibleSessionSkills(cfg bridgeconfig.Config, sess *session.Session) {
	if sess == nil {
		return
	}
	sess.PruneInvisibleDynamicSkills(bridgeruntime.VisibleSkillNames(cfg))
}

func RecentMessages(history *agent.History, limit int) []llm.Message {
	if history == nil || limit <= 0 {
		return nil
	}
	messages := history.Messages()
	filtered := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser, llm.RoleAssistant:
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) <= limit {
		return llm.CloneMessages(filtered)
	}
	return llm.CloneMessages(filtered[len(filtered)-limit:])
}

func ToolCatalogNames(catalog tools.ToolCatalog) []string {
	if catalog == nil {
		return nil
	}
	defs := catalog.ToolDefs()
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if name := strings.TrimSpace(def.Name); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func AutoResumePendingHumanTools(
	ctx context.Context,
	registry *tools.Registry,
	sess *session.Session,
	traceID string,
) error {
	if registry == nil || sess == nil || len(sess.HumanAnswers) == 0 {
		return nil
	}
	for _, questionID := range sortedAnsweredQuestionIDs(sess.HumanAnswers) {
		if err := resumeAnsweredHumanTool(ctx, registry, sess, questionID, traceID); err != nil {
			return err
		}
	}
	return nil
}

func resumeAnsweredHumanTool(
	ctx context.Context,
	registry *tools.Registry,
	sess *session.Session,
	questionID string,
	traceID string,
) error {
	question, ok := sess.PendingQuestions[questionID]
	if !ok {
		return nil
	}
	resumer := resolveHumanAnswerResumer(registry, question.ToolName)
	if resumer == nil {
		return nil
	}
	toolCtx := tools.WithToolCallID(ctx, question.ToolCallID)
	answer := sess.HumanAnswers[questionID]
	output, meta, handled, err := resumer.ResumeFromHumanAnswer(toolCtx, questionID, answer, traceID)
	if err != nil {
		return err
	}
	if !handled {
		return nil
	}
	item, ok := sess.ConsumeAnsweredQuestion(questionID)
	if !ok {
		return nil
	}
	if meta.AwaitingHuman != nil {
		return awaitingHumanError(meta.AwaitingHuman)
	}
	sess.AddMessage(sessionturn.AgentMessageForResolvedHumanTool(
		item.Question.ToolCallID,
		item.Question.ToolName,
		item.Question.TraceID,
		output,
	))
	return nil
}

func awaitingHumanError(payload *tools.AwaitingHumanSignal) error {
	return &agent.ErrAwaitingHuman{
		QuestionID:    strings.TrimSpace(payload.QuestionID),
		Prompt:        strings.TrimSpace(payload.Prompt),
		SelectionMode: strings.TrimSpace(payload.SelectionMode),
		Options:       append([]tools.AskHumanOption(nil), payload.Options...),
	}
}

func resolveHumanAnswerResumer(
	registry *tools.Registry,
	toolName string,
) tools.HumanAnswerAutoResumer {
	if registry == nil {
		return nil
	}
	tool := registry.Get(strings.TrimSpace(toolName))
	if tool == nil {
		return nil
	}
	resumer, _ := tool.(tools.HumanAnswerAutoResumer)
	return resumer
}

func sortedAnsweredQuestionIDs(answers map[string]string) []string {
	if len(answers) == 0 {
		return nil
	}
	ids := make([]string, 0, len(answers))
	for questionID := range answers {
		if trimmed := strings.TrimSpace(questionID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	sort.Strings(ids)
	return ids
}
