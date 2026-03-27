package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/session"
)

const noDynamicToolsLoaded = "- No dynamic tools loaded."

func formatDynamicToolState(sess *session.Session, idleTurns int) string {
	if sess == nil {
		return noDynamicToolsLoaded
	}

	loads := sess.DynamicToolLoadsSnapshot()
	if len(loads) == 0 {
		return noDynamicToolsLoaded
	}

	lines := make([]string, 0, len(loads))
	for _, load := range loads {
		if strings.TrimSpace(load.ToolName) == "" {
			continue
		}
		lines = append(lines, dynamicToolStateLine(load, sess.TurnIndex, idleTurns))
	}
	if len(lines) == 0 {
		return noDynamicToolsLoaded
	}
	return strings.Join(lines, "\n")
}

func dynamicToolStateLine(load session.DynamicToolLoad, currentTurn int, idleTurns int) string {
	name := fmt.Sprintf("`%s`", strings.TrimSpace(load.ToolName))

	switch {
	case load.LoadedAtTurn == currentTurn:
		return fmt.Sprintf("- %s was loaded in this user turn and is available now.", name)
	case load.ExpiredAtTurn(currentTurn, idleTurns):
		return fmt.Sprintf("- %s is expired and must be loaded again with `tfind(action=\"load\")`.", name)
	case load.VisibleForTurn(currentTurn):
		return fmt.Sprintf(
			"- %s is active in this session; remaining_idle_turns=%d.",
			name,
			load.RemainingIdleTurns(currentTurn, idleTurns),
		)
	default:
		return fmt.Sprintf("- %s is pending for a future turn.", name)
	}
}
