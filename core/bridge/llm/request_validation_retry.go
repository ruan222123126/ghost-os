package llm

import "errors"

func IsRetryableCompletionProtocolError(err error) bool {
	return errors.Is(err, errAssistantReasoningReplayMissing)
}
