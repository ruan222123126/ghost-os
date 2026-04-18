package orchestration

import (
	"context"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func autoResumePendingHumanTools(
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
	sess.AddMessage(agentMessageForResolvedHumanTool(
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
