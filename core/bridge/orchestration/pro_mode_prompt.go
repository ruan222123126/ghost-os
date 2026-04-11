package orchestration

import (
	"time"

	bridgeconfig "ghost-os/bridge/config"
	bridgemode "ghost-os/bridge/mode"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func newProModeCatalog(base tools.ToolCatalog, allowComplete bool) tools.ToolCatalog {
	return bridgemode.NewCatalog(base, allowComplete)
}

func buildProModeSystemPrompt(cfg bridgeconfig.Config, catalog tools.ToolCatalog, request proModeRequest) (string, error) {
	basePrompt, err := buildSystemPromptForCatalog(cfg, catalog)
	if err != nil {
		return "", err
	}
	return bridgemode.BuildProSystemPrompt(basePrompt, request), nil
}

func buildProModeUserPrompt(request proModeRequest, records []session.IterationRecord, iteration int) string {
	return bridgemode.BuildUserPrompt(request, records, iteration)
}

func buildProModeMaxLimitMessage(request proModeRequest, records []session.IterationRecord) string {
	return bridgemode.BuildMaxLimitMessage(request, records)
}

func buildAgentIterationRecords(records []session.IterationRecord) []agentIterationSummaryItem {
	if len(records) == 0 {
		return nil
	}
	out := make([]agentIterationSummaryItem, 0, len(records))
	for _, record := range records {
		out = append(out, agentIterationSummaryItem{
			Iteration:      record.Iteration,
			Did:            record.Did,
			Remaining:      record.Remaining,
			Completed:      record.Completed,
			TraceID:        record.TraceID,
			RecordedAt:     record.RecordedAt.Format(time.RFC3339),
			FinalChangeLog: record.FinalChangeLog,
		})
	}
	return out
}

func cloneIterationRecords(runtime *session.IterationRuntime) []session.IterationRecord {
	if runtime == nil || len(runtime.Records) == 0 {
		return nil
	}
	out := make([]session.IterationRecord, len(runtime.Records))
	copy(out, runtime.Records)
	return out
}
