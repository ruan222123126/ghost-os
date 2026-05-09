package orchestration

import (
	"context"
	"strings"
)

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func selectGroupFailure(results []orchestrationMemberResult) *orchestrationMemberResult {
	for index := range results {
		if results[index].Status == taskRunStatusAwaitingHuman {
			return &results[index]
		}
	}
	for index := range results {
		if results[index].Status == taskRunStatusError && results[index].Error != context.Canceled.Error() {
			return &results[index]
		}
	}
	for index := range results {
		if results[index].Status != taskRunStatusSuccess {
			return &results[index]
		}
	}
	return nil
}

func cloneGroupSessionIDs(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
