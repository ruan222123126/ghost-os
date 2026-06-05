package orchestration

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type relayModeCatalog struct {
	base   tools.ToolCatalog
	extra  map[string]tools.Tool
	hidden map[string]bool
}

func newRelayModeCatalog(base tools.ToolCatalog, allowComplete bool) tools.ToolCatalog {
	extra := map[string]tools.Tool{
		"relay_update_record": tools.NewRelayUpdateRecordTool(),
	}
	if allowComplete {
		extra["relay_complete"] = tools.NewRelayCompleteTool()
	}
	return &relayModeCatalog{
		base:  base,
		extra: extra,
		hidden: map[string]bool{
			"ask_human": true,
		},
	}
}

func (c *relayModeCatalog) Get(name string) tools.Tool {
	if c == nil {
		return nil
	}
	trimmed := strings.TrimSpace(name)
	if c.hidden[trimmed] {
		return nil
	}
	if tool, ok := c.extra[trimmed]; ok {
		return tool
	}
	if c.base == nil {
		return nil
	}
	return c.base.Get(trimmed)
}

func (c *relayModeCatalog) ToolDefs() []llm.ToolDef {
	if c == nil {
		return nil
	}
	defs := relayVisibleToolDefs(c.base, c.hidden)
	for _, name := range sortedRelayToolNames(c.extra) {
		defs = append(defs, tools.ToolDefFromTool(c.extra[name]))
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func relayVisibleToolDefs(base tools.ToolCatalog, hidden map[string]bool) []llm.ToolDef {
	if base == nil {
		return nil
	}
	defs := make([]llm.ToolDef, 0)
	for _, def := range base.ToolDefs() {
		if !hidden[strings.TrimSpace(def.Name)] {
			defs = append(defs, def)
		}
	}
	return defs
}

func sortedRelayToolNames(extra map[string]tools.Tool) []string {
	names := make([]string, 0, len(extra))
	for name := range extra {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

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

func buildRelayModeProtocolErrorPrompt(errorText string, previousOutput string, relay TaskRelayConfig) string {
	requiredTool := "`relay_update_record`"
	if relay.StopPolicy == taskRelayStopPolicyAIDecides {
		requiredTool = "`relay_update_record` or `relay_complete`"
	}
	output := strings.TrimSpace(previousOutput)
	if output == "" {
		output = "(empty response)"
	}
	return strings.TrimSpace(fmt.Sprintf(
		"Ghost-OS relay controller error:\n%s\n\nYour last correction still ended without the required relay handoff tool.\n\nVisible output from this invalid round:\n%s\n\nCorrect the protocol error now by calling %s.\nRules:\n- Treat this as a controller error correction, not a user-facing answer.\n- Reuse the actual progress from this round; do not invent work.\n- Do not answer in plain text.",
		strings.TrimSpace(errorText),
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

func buildRelayAgentNodeResult(
	input relayAgentNodeResultInput,
) bridgeTasks.RunNodeResult {
	record := buildAgentMessageNodeResult(input.task, input.execution, taskNodeResultTimestamps{
		startedAt:  input.startedAt,
		finishedAt: input.finishedAt,
	})
	record.Output = relayAgentNodeOutput(input.execution, input.result)
	record.Preview = strings.TrimSpace(input.execution.ResponsePreview)
	record.Error = strings.TrimSpace(input.execution.Error)
	return record
}

type relayAgentNodeResultInput struct {
	task       ScheduledTask
	execution  bridgeTasks.ExecutionResult
	result     relayModeResult
	startedAt  time.Time
	finishedAt time.Time
}

func relayAgentNodeOutput(execution bridgeTasks.ExecutionResult, result relayModeResult) map[string]any {
	output := map[string]any{
		"session_id_output": strings.TrimSpace(execution.SessionIDOutput),
		"response_preview":  strings.TrimSpace(execution.ResponsePreview),
		"stopped_by":        strings.TrimSpace(result.StoppedBy),
		"relay_records":     relayRecordSnapshots(result.Records),
	}
	if strings.TrimSpace(execution.Error) != "" {
		output["error"] = strings.TrimSpace(execution.Error)
	}
	return output
}

func relayRecordSnapshots(records []session.RelayRecord) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, relayRecordSnapshot(record))
	}
	return out
}

func relayRecordSnapshot(record session.RelayRecord) map[string]any {
	return map[string]any{
		"round":           record.Round,
		"did":             strings.TrimSpace(record.Did),
		"remaining":       strings.TrimSpace(record.Remaining),
		"failed_attempts": append([]string(nil), record.FailedAttempts...),
		"next_step":       strings.TrimSpace(record.NextStep),
		"completed":       record.Completed,
		"trace_id":        strings.TrimSpace(record.TraceID),
	}
}
