package relay

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

const (
	StatusCompleted  = "completed"
	StatusIncomplete = "incomplete"
	StatusCancelled  = "cancelled"
	StatusError      = "error"
	StopCompleted    = "relay_complete"
	StopMaxRounds    = "max_rounds"
	StopCancelled    = "cancelled"
	StopError        = "error"
)

type RuntimeDependencies struct {
	Config               bridgeconfig.Config
	Client               agent.Completer
	Registry             *tools.Registry
	SystemPrompt         string
	SystemPromptOverride bool
	Cleanup              func()
}

func (d RuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type RuntimeBuilder interface {
	Build(runtimeOverrides *taskdefs.TaskRuntimeOverrides) (RuntimeDependencies, error)
}

type RunRegistry interface {
	Register(sessionID string, traceID string, cancel context.CancelFunc) error
	Unregister(sessionID string)
}

type Runner struct {
	RuntimeBuilder RuntimeBuilder
	SessionStore   *session.Store
	RunRegistry    RunRegistry
}

type Result struct {
	Message        string
	StoppedBy      string
	FinalChangeLog string
	Records        []session.RelayRecord
	SessionID      string
}

func (r Runner) ExecuteTask(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
) (Result, error) {
	if task.Relay == nil {
		return Result{}, errors.New("relay config is required")
	}
	if r.RuntimeBuilder == nil {
		return Result{}, errors.New("relay runtime builder is not configured")
	}
	deps, err := r.RuntimeBuilder.Build(task.RuntimeOverrides)
	if err != nil {
		return Result{}, err
	}
	defer deps.Close()
	relay := *task.Relay
	sess, err := loadOrCreateRelaySession(r.SessionStore, task.SessionID, deps.SystemPrompt)
	if err != nil {
		return Result{}, err
	}
	if err := r.saveRelaySessionShell(sess, task.SessionID); err != nil {
		return Result{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return Result{}, err
	}
	defer cleanup()
	return r.Run(tools.WithSessionCheckpoint(tools.WithSession(execCtx, sess), r.SessionStore), RunRequest{
		Deps:    deps,
		Relay:   relay,
		Task:    task,
		Session: sess,
		TraceID: traceID,
	})
}

type RunRequest struct {
	Deps    RuntimeDependencies
	Relay   taskdefs.TaskRelayConfig
	Task    taskdefs.ScheduledTask
	Session *session.Session
	TraceID string
}

func loadOrCreateRelaySession(
	store *session.Store,
	sessionID string,
	systemPrompt string,
) (*session.Session, error) {
	if store == nil {
		return nil, errors.New("session store is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.NewSession(systemPrompt), nil
	}
	return store.Load(strings.TrimSpace(sessionID))
}

func (r Runner) saveRelaySessionShell(sess *session.Session, taskSessionID string) error {
	if sess == nil || strings.TrimSpace(taskSessionID) != "" {
		return nil
	}
	return r.SessionStore.Save(sess)
}

func (r Runner) register(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
	if r.RunRegistry == nil {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := r.RunRegistry.Register(sessionID, traceID, cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		cancel()
		r.RunRegistry.Unregister(sessionID)
	}, nil
}

func (r Runner) Run(ctx context.Context, req RunRequest) (Result, error) {
	req.Session.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.TrimSpace(req.Task.Message)})
	req.Session.StartRelayRuntime(req.Task.Message, req.Relay.StopPolicy, req.Relay.MaxRounds, req.Relay.ExecutionTimeoutMS)
	if err := r.SessionStore.Save(req.Session); err != nil {
		return Result{}, err
	}
	result, runErr := r.runRounds(ctx, req)
	if runErr != nil {
		if err := r.finishErroredRun(req.Session, runErr); err != nil {
			return Result{}, err
		}
		return Result{}, runErr
	}
	req.Session.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: result.Message})
	req.Session.FinishRelayRuntime(relayResultStatus(result.StoppedBy), result.StoppedBy, result.Message, result.FinalChangeLog)
	if err := r.SessionStore.Save(req.Session); err != nil {
		return Result{}, err
	}
	result.SessionID = req.Session.ID
	return result, nil
}

func (r Runner) finishErroredRun(sess *session.Session, runErr error) error {
	status := StatusError
	stoppedBy := StopError
	if errors.Is(runErr, context.Canceled) {
		status = StatusCancelled
		stoppedBy = StopCancelled
	}
	sess.FinishRelayRuntime(status, stoppedBy, "", "")
	return r.SessionStore.Save(sess)
}

func finishRelayRoundCardError(
	ctx context.Context,
	handle *runcardHandle,
	sessionID string,
	output relayRoundRunOutput,
	err error,
) {
	if handle == nil {
		return
	}
	finalText := strings.TrimSpace(output.finalText)
	_ = handle.Finish(ctx, runcardFinishInput{
		status:          relayRoundStatusFromError(ctx, err),
		preview:         sharedtext.TruncateRunes(finalText, taskdefs.MaxResponsePreviewRunes),
		errorText:       err.Error(),
		finalText:       finalText,
		sourceSessionID: sessionID,
		finishedAt:      time.Now().UTC(),
	})
}

func finishRelayRoundCard(
	ctx context.Context,
	handle *runcardHandle,
	sessionID string,
	status string,
	summary string,
	errorText string,
) error {
	if handle == nil {
		return nil
	}
	return handle.Finish(ctx, runcardFinishInput{
		status:          status,
		preview:         summary,
		errorText:       errorText,
		finalText:       summary,
		sourceSessionID: sessionID,
		finishedAt:      time.Now().UTC(),
	})
}

func runRelayRoundProtocolErrorCorrection(
	ctx context.Context,
	repairAgent *agent.Agent,
	round int,
	traceID string,
	relay taskdefs.TaskRelayConfig,
	previousOutputs []string,
	sink streaming.Sink,
) (relayRoundRunOutput, error) {
	protocolErr := relayMissingHandoffError(round)
	correctionPrompt := buildRelayModeProtocolErrorPrompt(
		protocolErr.Error(),
		relayVisibleOutputs(previousOutputs...),
		relay,
	)
	correctionOutput, correctionErr := repairAgent.RunMessageStreamWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: correctionPrompt,
	}, traceID, sink)
	visibleOutput := relayVisibleOutputs(append(previousOutputs, correctionOutput)...)
	handoffErr, err := relayHandoffFromRunErr(correctionErr)
	if err != nil {
		return relayRoundRunOutput{finalText: visibleOutput}, err
	}
	if handoffErr == nil {
		return relayRoundRunOutput{finalText: visibleOutput}, protocolErr
	}
	return relayRoundRunOutput{handoff: handoffErr, traceID: traceID}, nil
}

func relayVisibleOutput(output string, repairOutput string) string {
	return relayVisibleOutputs(output, repairOutput)
}

func relayVisibleOutputs(outputs ...string) string {
	for i := len(outputs) - 1; i >= 0; i-- {
		if trimmed := strings.TrimSpace(outputs[i]); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func relayResultStatus(stoppedBy string) string {
	switch stoppedBy {
	case StopCompleted:
		return StatusCompleted
	case StopMaxRounds:
		return StatusIncomplete
	case StopCancelled:
		return StatusCancelled
	default:
		return StatusError
	}
}

func cloneRelayRecords(runtime *session.RelayRuntime) []session.RelayRecord {
	if runtime == nil || len(runtime.Records) == 0 {
		return nil
	}
	out := make([]session.RelayRecord, len(runtime.Records))
	copy(out, runtime.Records)
	return out
}

func sessionLifecyclePayloadBuilder(sessionID string) agent.StreamLifecyclePayloadBuilder {
	return internaltrace.NewSessionStreamLifecyclePayloadBuilder(
		func() string { return sessionID },
		func(response string) (string, bool, error) {
			return response, false, nil
		},
	)
}
