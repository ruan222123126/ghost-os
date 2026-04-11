package mode

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type ProCatalog struct {
	base   tools.ToolCatalog
	extra  map[string]tools.Tool
	hidden map[string]bool
}

func NewCatalog(base tools.ToolCatalog, allowComplete bool) tools.ToolCatalog {
	extra := map[string]tools.Tool{
		"pro_update_record": tools.NewProUpdateRecordTool(),
	}
	if allowComplete {
		extra["pro_complete"] = tools.NewProCompleteTool()
	}
	return &ProCatalog{
		base:  base,
		extra: extra,
		hidden: map[string]bool{
			"ask_human": true,
		},
	}
}

func (c *ProCatalog) Get(name string) tools.Tool {
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

func (c *ProCatalog) ToolDefs() []llm.ToolDef {
	if c == nil {
		return nil
	}

	defs := collectVisibleToolDefs(c.base, c.hidden)
	names := sortedProToolNames(c.extra)
	for _, name := range names {
		defs = append(defs, tools.ToolDefFromTool(c.extra[name]))
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func BuildProSystemPrompt(basePrompt string, request ProRequest) string {
	modeRule := "End every iteration by calling `pro_update_record`; never end with plain text."
	if request.Mode == Pro {
		modeRule = "End every iteration by calling `pro_update_record`, or call `pro_complete` only when the task is truly complete."
	}
	return strings.TrimSpace(basePrompt + "\n\n" + strings.Join([]string{
		"You are a fresh-memory Ghost-OS iteration worker.",
		"You do not retain any memory across iterations except the injected iteration records below.",
		"Do not ask the user for input. `ask_human` is intentionally unavailable in this mode.",
		modeRule,
		"The handoff record must stay concise: only what you did and what remains.",
	}, "\n"))
}

func BuildUserPrompt(request ProRequest, records []session.IterationRecord, iteration int) string {
	limitLine := buildLimitLine(request)
	history := buildHistory(records)
	return strings.TrimSpace(fmt.Sprintf(
		"Mode: %s\nIteration: %d\n%s\n\nOriginal task:\n%s\n\nPrevious iteration records:\n%s\n\nRules:\n- You are a fresh-memory worker; rely only on the task above and the iteration records in this prompt.\n- Make real repo progress.\n- End this iteration by calling the required pro tool; do not stop with plain text.",
		request.Mode,
		iteration,
		limitLine,
		request.OriginalTask,
		history,
	))
}

func BuildMaxLimitMessage(request ProRequest, records []session.IterationRecord) string {
	if len(records) == 0 {
		return fmt.Sprintf("Reached %s max_iterations=%d before any valid iteration record was produced.", request.Mode, request.MaxIterations)
	}
	last := records[len(records)-1]
	return fmt.Sprintf(
		"Reached %s max_iterations=%d without completion.\nLast completed work: %s\nRemaining work: %s",
		request.Mode,
		request.MaxIterations,
		last.Did,
		last.Remaining,
	)
}

func collectVisibleToolDefs(base tools.ToolCatalog, hidden map[string]bool) []llm.ToolDef {
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

func sortedProToolNames(extra map[string]tools.Tool) []string {
	names := make([]string, 0, len(extra))
	for name := range extra {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildLimitLine(request ProRequest) string {
	if request.Mode == Pro {
		return fmt.Sprintf("Max iterations: %d. Only `pro_complete` can stop the run early.", request.MaxIterations)
	}
	if request.MaxIterations > 0 {
		return fmt.Sprintf("Max iterations: %d. You still cannot stop early on your own.", request.MaxIterations)
	}
	return "Max iterations: unlimited until the user stops you or an error occurs."
}

func buildHistory(records []session.IterationRecord) string {
	if len(records) == 0 {
		return "(none yet)"
	}

	var history strings.Builder
	for _, record := range records {
		history.WriteString(fmt.Sprintf("%d. did: %s\n", record.Iteration, record.Did))
		history.WriteString(fmt.Sprintf("   remaining: %s\n", record.Remaining))
		if strings.TrimSpace(record.FinalChangeLog) != "" {
			history.WriteString(fmt.Sprintf("   final_change_log: %s\n", record.FinalChangeLog))
		}
	}
	return strings.TrimSpace(history.String())
}
