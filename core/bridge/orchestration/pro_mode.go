package orchestration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

const (
	proModePro              = "pro"
	proModeProx             = "prox"
	proModeStatusRunning    = "running"
	proModeStatusCompleted  = "completed"
	proModeStatusIncomplete = "incomplete"
	proModeStatusCancelled  = "cancelled"
	proModeStatusError      = "error"
	proModeStopCompleted    = "pro_complete"
	proModeStopMaxLimit     = "max_iterations"
	proModeStopCancelled    = "cancelled"
	proModeStopError        = "error"
)

type proModeRequest struct {
	Mode          string
	OriginalTask  string
	MaxIterations int
	Unlimited     bool
}

type proModeResult struct {
	Message        string
	StoppedBy      string
	FinalChangeLog string
	Records        []session.IterationRecord
}

type proModeCatalog struct {
	base   tools.ToolCatalog
	extra  map[string]tools.Tool
	hidden map[string]bool
}

func parseProModeRequest(message string, defaultMaxIterations int) (proModeRequest, bool, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return proModeRequest{}, false, nil
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return proModeRequest{}, false, nil
	}

	mode := strings.ToLower(strings.TrimSpace(fields[0]))
	switch mode {
	case proModePro, proModeProx:
	default:
		return proModeRequest{}, false, nil
	}

	index := 1
	maxIterations := 0
	if index < len(fields) {
		if parsed, err := strconv.Atoi(fields[index]); err == nil {
			if parsed <= 0 {
				return proModeRequest{}, true, errors.New("pro/prox max iterations must be > 0")
			}
			maxIterations = parsed
			index++
		}
	}

	task := strings.TrimSpace(strings.Join(fields[index:], " "))
	if task == "" {
		return proModeRequest{}, true, errors.New("pro/prox task is required")
	}

	req := proModeRequest{
		Mode:         mode,
		OriginalTask: task,
	}
	if mode == proModePro {
		if maxIterations <= 0 {
			maxIterations = defaultMaxIterations
		}
		req.MaxIterations = maxIterations
		return req, true, nil
	}

	req.Unlimited = maxIterations == 0
	req.MaxIterations = maxIterations
	return req, true, nil
}

func (s *bridgeService) executeProModeAction(ctx context.Context, prepared preparedAgentTurnRequest, traceID string) (agentResponse, int, error) {
	factory := s.runtimeFactory
	if factory == nil {
		factory = newAgentRuntimeFactoryWithTaskManager(s.taskToolManager())
	}
	deps, err := factory.Build(s.configStore)
	if err != nil {
		return agentResponse{}, 0, err
	}
	defer deps.Close()

	request, matched, err := parseProModeRequest(prepared.message, deps.cfg.ProMaxIterations)
	if err != nil {
		return agentResponse{}, 0, err
	}
	if !matched {
		return agentResponse{}, 0, errors.New("pro mode not requested")
	}

	store, code, err := s.requireSessionStore()
	if err != nil {
		return agentResponse{}, code, err
	}

	sess, err := loadOrCreateIterationSession(store, prepared.sessionID, deps.systemPrompt)
	if err != nil {
		return agentResponse{}, mapSessionStorageError(err), err
	}

	execCtx, cleanup, err := s.registerProModeRun(ctx, sess.ID, traceID)
	if err != nil {
		return agentResponse{}, http.StatusConflict, err
	}
	defer cleanup()

	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.TrimSpace(prepared.message)})
	sess.StartIterationRuntime(request.Mode, request.OriginalTask, request.MaxIterations, request.Unlimited)
	if err := store.Save(sess); err != nil {
		return agentResponse{}, mapSessionStorageError(err), err
	}

	result, runErr := s.runProModeIterations(tools.WithSession(execCtx, sess), deps, sess, request, traceID)
	if runErr != nil {
		status := proModeStatusError
		stoppedBy := proModeStopError
		if errors.Is(runErr, context.Canceled) {
			status = proModeStatusCancelled
			stoppedBy = proModeStopCancelled
		}
		sess.FinishIterationRuntime(status, stoppedBy, "", "")
		_ = store.Save(sess)
		normalizedErr, statusCode := normalizeAgentExecutionError(runErr)
		return agentResponse{}, statusCode, normalizedErr
	}

	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: result.Message})
	sess.FinishIterationRuntime(proModeResultStatus(result.StoppedBy), result.StoppedBy, result.Message, result.FinalChangeLog)
	if err := store.Save(sess); err != nil {
		return agentResponse{}, mapSessionStorageError(err), err
	}

	payload, err := newAgentResponsePayload(result.Message, sess.ID, nil, agentResponseMeta{
		Mode:             request.Mode,
		IterationCount:   len(result.Records),
		StoppedBy:        result.StoppedBy,
		FinalChangeLog:   result.FinalChangeLog,
		IterationSummary: buildAgentIterationRecords(result.Records),
	})
	if err != nil {
		return agentResponse{}, http.StatusInternalServerError, err
	}
	return payload, http.StatusOK, nil
}

func loadOrCreateIterationSession(store *session.Store, sessionID string, systemPrompt string) (*session.Session, error) {
	if store == nil {
		return nil, errors.New("session store is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.NewSession(systemPrompt), nil
	}
	return store.Load(strings.TrimSpace(sessionID))
}

func (s *bridgeService) registerProModeRun(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
	if s == nil || s.runRegistry == nil {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := s.runRegistry.Register(sessionID, traceID, cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		cancel()
		s.runRegistry.Unregister(sessionID)
	}, nil
}

func (s *bridgeService) runProModeIterations(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	request proModeRequest,
	traceID string,
) (proModeResult, error) {
	baseCatalog := newToolSelectionPolicy(deps.cfg.ToolSelector).scopeCatalog(deps.registry)
	catalog := newProModeCatalog(baseCatalog, request.Mode == proModePro)
	systemPrompt, err := buildProModeSystemPrompt(deps.cfg, catalog, request)
	if err != nil {
		return proModeResult{}, err
	}

	for iteration := 1; ; iteration++ {
		if err := ctx.Err(); err != nil {
			return proModeResult{}, err
		}
		if !request.Unlimited && request.MaxIterations > 0 && iteration > request.MaxIterations {
			records := cloneIterationRecords(sess.IterationRuntime)
			return proModeResult{
				Message:        buildProModeMaxLimitMessage(request, records),
				StoppedBy:      proModeStopMaxLimit,
				FinalChangeLog: "",
				Records:        records,
			}, nil
		}

		history := agent.NewHistory(systemPrompt)
		turnAgent := agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns)
		iterationTraceID := fmt.Sprintf("%s-pro-%d", strings.TrimSpace(traceID), iteration)
		_, runErr := turnAgent.RunWithTraceID(ctx, buildProModeUserPrompt(request, cloneIterationRecords(sess.IterationRuntime), iteration), iterationTraceID)
		if runErr == nil {
			return proModeResult{}, fmt.Errorf("pro iteration %d ended without pro_update_record or pro_complete", iteration)
		}

		var handoffErr *agent.ErrIterationHandoff
		if errors.As(runErr, &handoffErr) {
			record := session.IterationRecord{
				Iteration:      iteration,
				Did:            handoffErr.Did,
				Remaining:      handoffErr.Remaining,
				Completed:      handoffErr.Completed,
				TraceID:        iterationTraceID,
				RecordedAt:     time.Now().UTC(),
				FinalChangeLog: handoffErr.FinalChangeLog,
			}
			sess.AppendIterationRecord(record)
			if s.sessionStore != nil {
				if err := s.sessionStore.Save(sess); err != nil {
					return proModeResult{}, err
				}
			}
			if handoffErr.Completed {
				return proModeResult{
					Message:        handoffErr.FinalMessage,
					StoppedBy:      proModeStopCompleted,
					FinalChangeLog: handoffErr.FinalChangeLog,
					Records:        cloneIterationRecords(sess.IterationRuntime),
				}, nil
			}
			continue
		}

		return proModeResult{}, runErr
	}
}

func newProModeCatalog(base tools.ToolCatalog, allowComplete bool) tools.ToolCatalog {
	extra := map[string]tools.Tool{
		"pro_update_record": tools.NewProUpdateRecordTool(),
	}
	if allowComplete {
		extra["pro_complete"] = tools.NewProCompleteTool()
	}
	return &proModeCatalog{
		base:  base,
		extra: extra,
		hidden: map[string]bool{
			"ask_human": true,
		},
	}
}

func (c *proModeCatalog) Get(name string) tools.Tool {
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

func (c *proModeCatalog) ToolDefs() []llm.ToolDef {
	if c == nil {
		return nil
	}
	defs := make([]llm.ToolDef, 0)
	if c.base != nil {
		for _, def := range c.base.ToolDefs() {
			if c.hidden[strings.TrimSpace(def.Name)] {
				continue
			}
			defs = append(defs, def)
		}
	}
	names := make([]string, 0, len(c.extra))
	for name := range c.extra {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		tool := c.extra[name]
		defs = append(defs, llm.ToolDef{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func buildProModeSystemPrompt(cfg Config, catalog tools.ToolCatalog, request proModeRequest) (string, error) {
	modeRule := "End every iteration by calling `pro_update_record`; never end with plain text."
	if request.Mode == proModePro {
		modeRule = "End every iteration by calling `pro_update_record`, or call `pro_complete` only when the task is truly complete."
	}
	basePrompt, err := buildSystemPromptForCatalog(cfg, catalog)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(basePrompt + "\n\n" + strings.Join([]string{
		"You are a fresh-memory Ghost-OS iteration worker.",
		"You do not retain any memory across iterations except the injected iteration records below.",
		"Do not ask the user for input. `ask_human` is intentionally unavailable in this mode.",
		modeRule,
		"The handoff record must stay concise: only what you did and what remains.",
	}, "\n")), nil
}

func buildProModeUserPrompt(request proModeRequest, records []session.IterationRecord, iteration int) string {
	limitLine := "Max iterations: unlimited until the user stops you or an error occurs."
	if request.Mode == proModePro {
		limitLine = fmt.Sprintf("Max iterations: %d. Only `pro_complete` can stop the run early.", request.MaxIterations)
	} else if request.MaxIterations > 0 {
		limitLine = fmt.Sprintf("Max iterations: %d. You still cannot stop early on your own.", request.MaxIterations)
	}

	var history strings.Builder
	if len(records) == 0 {
		history.WriteString("(none yet)")
	} else {
		for _, record := range records {
			history.WriteString(fmt.Sprintf("%d. did: %s\n", record.Iteration, record.Did))
			history.WriteString(fmt.Sprintf("   remaining: %s\n", record.Remaining))
			if strings.TrimSpace(record.FinalChangeLog) != "" {
				history.WriteString(fmt.Sprintf("   final_change_log: %s\n", record.FinalChangeLog))
			}
		}
	}

	return strings.TrimSpace(fmt.Sprintf(
		"Mode: %s\nIteration: %d\n%s\n\nOriginal task:\n%s\n\nPrevious iteration records:\n%s\n\nRules:\n- You are a fresh-memory worker; rely only on the task above and the iteration records in this prompt.\n- Make real repo progress.\n- End this iteration by calling the required pro tool; do not stop with plain text.",
		request.Mode,
		iteration,
		limitLine,
		request.OriginalTask,
		strings.TrimSpace(history.String()),
	))
}

func buildProModeMaxLimitMessage(request proModeRequest, records []session.IterationRecord) string {
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

func buildAgentIterationRecords(records []session.IterationRecord) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, map[string]any{
			"iteration":        record.Iteration,
			"did":              record.Did,
			"remaining":        record.Remaining,
			"completed":        record.Completed,
			"trace_id":         record.TraceID,
			"recorded_at":      record.RecordedAt.Format(time.RFC3339),
			"final_change_log": record.FinalChangeLog,
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

func proModeResultStatus(stoppedBy string) string {
	switch strings.TrimSpace(stoppedBy) {
	case proModeStopCompleted:
		return proModeStatusCompleted
	case proModeStopMaxLimit:
		return proModeStatusIncomplete
	case proModeStopCancelled:
		return proModeStatusCancelled
	default:
		return proModeStatusError
	}
}
