package orchestration

import "strings"

const ownerMissingDispatchMessage = "owner turn ended without orchestration_dispatch; orchestration cannot advance yet"

func ownerRunCardStatus(response string, runErr error) (string, string, string) {
	handoffErr, err := ownerDispatchHandoffFromRunErr(runErr)
	if err != nil {
		return taskRunStatusError, "", err.Error()
	}
	if handoffErr != nil {
		return taskRunStatusSuccess, strings.TrimSpace(handoffErr.Did), ""
	}
	if runErr != nil {
		return taskRunStatusError, "", runErr.Error()
	}
	preview := strings.TrimSpace(response)
	if preview == "" {
		return taskRunStatusError, "", ownerMissingDispatchMessage
	}
	return taskRunStatusError, preview, ownerMissingDispatchMessage
}
