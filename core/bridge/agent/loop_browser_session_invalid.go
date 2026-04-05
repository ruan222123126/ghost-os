package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const maxConsecutiveBrowserSessionInvalid = 2

func (state *agentRunState) updateBrowserSessionInvalidPolicy(
	ctx context.Context,
	turn int,
	stats toolCallTurnStats,
) error {
	if stats.browserSessionInvalidFailures <= 0 {
		state.consecutiveBrowserSessionInvalid = 0
		return nil
	}

	state.consecutiveBrowserSessionInvalid += stats.browserSessionInvalidFailures
	if state.consecutiveBrowserSessionInvalid < maxConsecutiveBrowserSessionInvalid {
		return nil
	}
	return state.terminalRunError(
		ctx,
		turn,
		repeatedBrowserSessionInvalidError(
			state.traceID,
			turn,
			state.consecutiveBrowserSessionInvalid,
			stats.browserSessionInvalidLastError,
		),
	)
}

func repeatedBrowserSessionInvalidError(
	traceID string,
	turn int,
	consecutive int,
	lastError string,
) error {
	payload := map[string]any{
		"code":                 "browser_session_invalid_repeated",
		"message":              "browser_control hit browser_session_invalid repeatedly; aborting run without fallback",
		"tool":                 "browser_control",
		"trace_id":             traceID,
		"turn":                 turn,
		"consecutive_failures": consecutive,
	}
	if trimmed := strings.TrimSpace(lastError); trimmed != "" {
		payload["last_error"] = trimmed
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"trace_id=%s turn=%d browser_session_invalid_repeated after %d failures",
			traceID,
			turn,
			consecutive,
		)
	}
	return errors.New(string(encoded))
}
