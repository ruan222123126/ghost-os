package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func appendGraphQLExecutionFeedback(history *History, output string) {
	if history == nil {
		return
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return
	}
	history.Append(llm.Message{
		Role: llm.RoleInternal,
		Text: formatGraphQLExecutionFeedback(trimmed),
	})
}

func formatGraphQLExecutionFeedback(output string) string {
	var decoded any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		return "[GRAPHQL_EXECUTION_RESULT]\n" + output
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "[GRAPHQL_EXECUTION_RESULT]\n" + output
	}
	return fmt.Sprintf("[GRAPHQL_EXECUTION_RESULT]\n%s", string(normalized))
}
