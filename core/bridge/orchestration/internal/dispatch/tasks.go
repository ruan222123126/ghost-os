package dispatch

import (
	"fmt"
	"strings"

	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
)

func NormalizeTaskListScope(scope string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(scope))
	switch normalized {
	case "", apptasks.ScopeUser:
		return apptasks.ScopeUser, nil
	case apptasks.ScopeSystem:
		return apptasks.ScopeSystem, nil
	case apptasks.ScopeOrchestration:
		return apptasks.ScopeOrchestration, nil
	default:
		return "", fmt.Errorf("invalid task scope %q", scope)
	}
}
