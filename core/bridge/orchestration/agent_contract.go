package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type agentResponseMeta struct {
	Mode string
}

type agentRuntimeDependencies struct {
	cfg                  bridgeconfig.Config
	client               agent.Completer
	registry             *tools.Registry
	systemPrompt         string
	systemPromptOverride bool
	systemPromptFiles    *bridgeconfig.SystemPromptFiles
	cleanup              func()
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

func (d agentRuntimeDependencies) Config() bridgeconfig.Config {
	return d.cfg
}

func (d agentRuntimeDependencies) Client() agent.Completer {
	return d.client
}

func (d agentRuntimeDependencies) Registry() *tools.Registry {
	return d.registry
}

func (d agentRuntimeDependencies) SystemPrompt() string {
	return d.systemPrompt
}

func (d agentRuntimeDependencies) SystemPromptOverride() bool {
	return d.systemPromptOverride
}

func (d agentRuntimeDependencies) SystemPromptFiles() *bridgeconfig.SystemPromptFiles {
	return d.systemPromptFiles
}

type AgentRuntimeFactory interface {
	Build(store bridgeconfig.Store) (agentRuntimeDependencies, error)
}

type RuntimeDependencies = agentRuntimeDependencies
type RuntimeCompleter = agent.Completer
type RuntimeToolRegistry = tools.Registry
type ServerConfig = bridgeconfig.ServerConfig
type MobileWebRTCConfig = bridgeconfig.MobileWebRTCConfig
type ConfigStore = bridgeconfig.Store

var errPreparedAgentStreamRunnerRequired = errors.New("prepared agent stream runner is not configured")

type PreparedAgentStream struct {
	run func(context.Context, streaming.Sink) (string, string, error)
}

func NewPreparedAgentStream(run func(context.Context, StreamSink) (string, string, error)) PreparedAgentStream {
	return PreparedAgentStream{run: run}
}

func (p PreparedAgentStream) Run(ctx context.Context, sink StreamSink) (string, string, error) {
	if p.run == nil {
		return "", "", bus.WrapError(ServiceErrorInternal, errPreparedAgentStreamRunnerRequired)
	}
	return p.run(ctx, sink)
}

func LoadServerConfig() (ServerConfig, error) {
	return bridgeconfig.LoadServerConfig()
}

func NewConfigStoreFromEnv() (ConfigStore, error) {
	return bridgeconfig.NewStoreFromEnv()
}

func NewServiceWithSessionStorePath(store ConfigStore, sessionsPath string) (*Service, error) {
	sessionStore, err := session.NewStore(sessionsPath, session.StoreOptions{
		HumanLogFullEnabled: func() bool {
			if store == nil {
				return false
			}
			return store.Snapshot().SessionHumanLogFullEnabled
		},
	})
	if err != nil {
		return nil, err
	}
	return NewService(store, sessionStore, nil), nil
}

func NewRuntimeDependencies(
	cfg bridgeconfig.Config,
	client RuntimeCompleter,
	registry *RuntimeToolRegistry,
	systemPrompt string,
	cleanup func(),
) RuntimeDependencies {
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       client,
		registry:     registry,
		systemPrompt: systemPrompt,
		cleanup:      cleanup,
	}
}

type runtimeFactoryAdapter struct {
	inner bridgeruntime.AgentRuntimeFactory
}

func (f runtimeFactoryAdapter) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	var innerStore *bridgeruntime.ConfigStore
	if store != nil {
		innerStore = bridgeruntime.WrapConfigStore(store)
	}
	deps, err := f.inner.Build(innerStore)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return agentRuntimeDependencies{
		cfg:          deps.Config(),
		client:       deps.Client(),
		registry:     deps.Registry(),
		systemPrompt: deps.SystemPrompt(),
		cleanup:      deps.Close,
	}, nil
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return runtimeFactoryAdapter{inner: bridgeruntime.NewAgentRuntimeFactory()}
}

// parseSessionEndSignal 只识别完整会话结束信号；普通文本/普通 JSON 均按普通回复返回。
func parseSessionEndSignal(raw string) (message string, signal *assistantSessionEndSignalPayload, err error) {
	return agentturn.ParseSessionEndSignal(raw)
}

func parseSessionEndForStream(response string) (string, bool, error) {
	normalized, sessionEnd, err := parseSessionEndSignal(response)
	return normalized, sessionEnd != nil, err
}

// newAgentResponsePayload 构造并校验 AGENT_SEND 成功响应，确保会话结束契约稳定。
func newAgentResponsePayload(message string, sessionID string, sessionEnd *assistantSessionEndSignalPayload, meta agentResponseMeta) (agentResponse, error) {
	return agentturn.NewResponsePayload(message, sessionID, sessionEnd, agentturn.ResponseMeta{Mode: meta.Mode})
}

// validateAgentResponsePayload 在跨进程返回前执行最小契约校验。
func validateAgentResponsePayload(payload agentResponse) error {
	return agentturn.ValidateResponsePayload(payload)
}

var errStructuredAgentRunnerRequired = errors.New("configured agent runner does not support image input")
var errRuntimeOverrideRunnerRequired = errors.New("configured agent runner does not support runtime_overrides")
var errRuntimeOverrideWithImages = errors.New("runtime_overrides do not support image input")
var errRequestRuntimeRunnerRequired = errors.New("configured agent runner does not support request runtime options")

func hasAgentInputImages(message llm.Message) bool {
	return agentturn.HasInputImages(message)
}

func (s *bridgeService) runPreparedAgentTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (string, string, error) {
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.RequestRuntime)
	if err != nil {
		return "", "", err
	}
	if prepared.RuntimeOverrides != nil {
		if hasAgentInputImages(prepared.UserInput) {
			return "", "", errRuntimeOverrideWithImages
		}
		runner, ok := runner.(SessionTurnRunnerWithOverrides)
		if !ok {
			return "", "", errRuntimeOverrideRunnerRequired
		}
		return runner.RunTurnWithOverrides(
			ctx,
			prepared.Message,
			prepared.SessionID,
			traceID,
			prepared.RuntimeOverrides,
		)
	}
	if hasAgentInputImages(prepared.UserInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnInput(ctx, prepared.UserInput, prepared.SessionID, traceID)
	}
	return runner.RunTurn(ctx, prepared.Message, prepared.SessionID, traceID)
}

func (s *bridgeService) runPreparedAgentTurnStream(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.RequestRuntime)
	if err != nil {
		return "", "", err
	}
	if prepared.RuntimeOverrides != nil {
		if hasAgentInputImages(prepared.UserInput) {
			return "", "", errRuntimeOverrideWithImages
		}
		runnerWithOverrides, ok := runner.(SessionTurnStreamRunnerWithOverrides)
		if !ok {
			return "", "", errRuntimeOverrideRunnerRequired
		}
		return runnerWithOverrides.RunTurnStreamWithOverrides(
			ctx,
			prepared.Message,
			prepared.SessionID,
			traceID,
			sink,
			prepared.RuntimeOverrides,
		)
	}
	if hasAgentInputImages(prepared.UserInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnStreamInput(ctx, prepared.UserInput, prepared.SessionID, traceID, sink)
	}
	return runner.RunTurnStream(ctx, prepared.Message, prepared.SessionID, traceID, sink)
}

func (s *bridgeService) prepareAgentStreamAction(
	ctx context.Context,
	params agentParams,
	traceID string,
) (PreparedAgentStream, ServiceResult, error) {
	if err := ctx.Err(); err != nil {
		return PreparedAgentStream{}, ServiceResult{}, bus.WrapError(ServiceErrorConflict, err)
	}
	prepared, err := s.agentTurnService().Prepare(params, params.RuntimeOverrides, traceID)
	if err != nil {
		return PreparedAgentStream{}, ServiceResult{}, err
	}
	return PreparedAgentStream{
		run: func(ctx context.Context, sink streaming.Sink) (string, string, error) {
			broadcastSink := internaltrace.NewSessionStreamBroadcastSink(sink, s.sessionPushHub())
			return s.agentTurnService().ExecutePreparedStream(ctx, prepared, traceID, broadcastSink)
		},
	}, bus.ResultSuccess(map[string]any{}), nil
}

func requestScopedAgentRunner(
	runner SessionTurnRunner,
	options *requestRuntimeOptions,
) (SessionTurnRunner, error) {
	if options == nil {
		return runner, nil
	}
	aware, ok := runner.(requestRuntimeAwareRunner)
	if !ok {
		return nil, errRequestRuntimeRunnerRequired
	}
	scoped, ok := aware.WithRequestRuntimeOptions(options).(SessionTurnRunner)
	if !ok {
		return nil, errRequestRuntimeRunnerRequired
	}
	return scoped, nil
}
