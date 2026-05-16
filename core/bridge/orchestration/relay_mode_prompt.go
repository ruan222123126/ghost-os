package orchestration

import (
	"fmt"
	"strings"

	"ghost-os/bridge/session"
)

func buildRelayModeSystemPrompt(basePrompt string, relay TaskRelayConfig) string {
	modeRule := "End every round by calling `relay_update_record`; do not end with plain text."
	if relay.StopPolicy == taskRelayStopPolicyAIDecides {
		modeRule = fmt.Sprintf(
			"End every round by calling `relay_update_record`, or call `relay_complete` only when the task is truly complete; this run force-stops at max_rounds=%d.",
			relay.MaxRounds,
		)
	}
	if relay.StopPolicy == taskRelayStopPolicyMaxRounds {
		modeRule = fmt.Sprintf(
			"End every round by calling `relay_update_record`; this run stops at max_rounds=%d unless the user stops it.",
			relay.MaxRounds,
		)
	}
	return strings.TrimSpace(basePrompt + "\n\n" + strings.Join([]string{
		"You are a fresh-memory Ghost-OS relay worker.",
		"You do not retain memory across rounds except the injected relay records below.",
		"Do not ask the user for input. `ask_human` is intentionally unavailable in relay mode.",
		modeRule,
		"Every relay record must include did, remaining, failed_attempts, and next_step.",
	}, "\n"))
}

func buildRelayModeUserPrompt(task string, records []session.RelayRecord, round int, relay TaskRelayConfig) string {
	return strings.TrimSpace(fmt.Sprintf(
		"Relay round: %d\nStop policy: %s\nMax rounds: %s\n\nOriginal task:\n%s\n\nPrevious relay records:\n%s\n\nRules:\n- You are a fresh-memory worker; rely only on the task above and the relay records in this prompt.\n- Make real progress with available tools.\n- End this round by calling the required relay tool.",
		round,
		relay.StopPolicy,
		relayMaxRoundsLine(relay),
		strings.TrimSpace(task),
		formatRelayHistory(records),
	))
}

func buildRelayModeRepairPrompt(previousOutput string, relay TaskRelayConfig) string {
	requiredTool := "`relay_update_record`"
	if relay.StopPolicy == taskRelayStopPolicyAIDecides {
		requiredTool = "`relay_update_record` or `relay_complete`"
	}
	output := strings.TrimSpace(previousOutput)
	if output == "" {
		output = "(empty response)"
	}
	return strings.TrimSpace(fmt.Sprintf(
		"Your previous reply was invalid for relay mode because it ended as plain text instead of using the required relay handoff tool.\n\nPrevious reply:\n%s\n\nRe-emit this round now by calling %s.\nRules:\n- Reuse the actual progress from this round; do not invent work.\n- Do not answer in plain text.",
		output,
		requiredTool,
	))
}

func buildRelayMaxRoundsMessage(task string, records []session.RelayRecord, maxRounds int) string {
	if len(records) == 0 {
		return fmt.Sprintf("Reached relay max_rounds=%d before any valid relay record was produced.", maxRounds)
	}
	last := records[len(records)-1]
	return fmt.Sprintf(
		"Reached relay max_rounds=%d without completion.\nTask: %s\nLast completed work: %s\nRemaining work: %s\nNext step: %s",
		maxRounds,
		strings.TrimSpace(task),
		last.Did,
		last.Remaining,
		last.NextStep,
	)
}

func relayMaxRoundsLine(relay TaskRelayConfig) string {
	return fmt.Sprintf("%d", relay.MaxRounds)
}

func formatRelayHistory(records []session.RelayRecord) string {
	if len(records) == 0 {
		return "(none yet)"
	}
	var out strings.Builder
	for _, record := range records {
		out.WriteString(formatRelayRecord(record))
	}
	return strings.TrimSpace(out.String())
}

func formatRelayRecord(record session.RelayRecord) string {
	return fmt.Sprintf(
		"%d. did: %s\n   remaining: %s\n   failed_attempts: %s\n   next_step: %s\n",
		record.Round,
		record.Did,
		record.Remaining,
		strings.Join(record.FailedAttempts, " | "),
		record.NextStep,
	)
}
