package agent

import (
	"errors"

	"ghost-os/bridge/llm"
)

func commitsPartialToolCallTurn(err error) bool {
	var awaitingErr *ErrAwaitingHuman
	if errors.As(err, &awaitingErr) {
		return true
	}

	var handoffErr *ErrIterationHandoff
	return errors.As(err, &handoffErr)
}

func shouldCommitEventEmitToolTurn(history *History) bool {
	if history == nil {
		return false
	}
	messages := history.Messages()
	if len(messages) == 0 {
		return false
	}
	return messages[len(messages)-1].Role == llm.RoleTool
}
