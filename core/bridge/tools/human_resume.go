package tools

import "context"

type HumanAnswerAutoResumer interface {
	ResumeFromHumanAnswer(
		ctx context.Context,
		questionID string,
		answer string,
		traceID string,
	) (string, ExecuteMeta, bool, error)
}
