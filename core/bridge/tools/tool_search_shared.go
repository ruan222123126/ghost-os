package tools

import "ghost-os/bridge/session"

func availableNextTurn(load session.DynamicToolLoad, currentTurn int, idleTurns int) bool {
	return !load.VisibleForTurn(currentTurn) &&
		!load.ExpiredAtTurn(currentTurn, idleTurns) &&
		load.LoadedAtTurn > currentTurn
}
