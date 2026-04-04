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
		question, ok := sess.PendingQuestions[questionID]
		if !ok {
			continue
		}
		resumer := resolveHumanAnswerResumer(registry, question.ToolName)
		if resumer == nil {
			continue
		}
		toolCtx := tools.WithToolCallID(ctx, question.ToolCallID)
		answer := sess.HumanAnswers[questionID]
		output, meta, handled, err := resumer.ResumeFromHumanAnswer(toolCtx, questionID, answer, traceID)
		if err != nil {
			return err
		}
		if !handled {
			continue
		}
		item, ok := sess.ConsumeAnsweredQuestion(questionID)
		if !ok {
			continue
		}
		if meta.AwaitingHuman != nil {
			return &agent.ErrAwaitingHuman{
				QuestionID:    strings.TrimSpace(meta.AwaitingHuman.QuestionID),
				Prompt:        strings.TrimSpace(meta.AwaitingHuman.Prompt),
				SelectionMode: strings.TrimSpace(meta.AwaitingHuman.SelectionMode),
				Options:       append([]tools.AskHumanOption(nil), meta.AwaitingHuman.Options...),
			}
		}
		sess.AddMessage(agentMessageForResolvedHumanTool(
			item.Question.ToolCallID,
			item.Question.ToolName,
			item.Question.TraceID,
			output,
		))
	}
	return nil
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
