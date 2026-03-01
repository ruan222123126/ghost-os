package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"runtime"
	"strconv"
	"strings"

	"ghost-os/bridge/agent"
	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       *llm.Client
	registry     *tools.Registry
	systemPrompt string
}

type agentSessionExecutor struct {
	deps          agentRuntimeDependencies
	sessionStore  *session.Store
	memoryManager *memory.MemoryManager
	traceID       string
}

const agentWarmMemoryCapacity = 100

// runAgent 组装最小可运行链路：配置 -> LLM 客户端 -> 工具目录 -> Agent。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	return runAgentWithConfigStore(ctx, userMessage, nil, "")
}

// runAgentWithConfigStore 允许注入配置存储与 trace id，便于服务层复用。
func runAgentWithConfigStore(ctx context.Context, userMessage string, store *ConfigStore, traceID string) (string, error) {
	deps, err := buildAgentRuntimeDependencies(store)
	if err != nil {
		return "", err
	}

	a := agent.NewAgent(deps.client, deps.registry, deps.systemPrompt, deps.cfg.MaxTurns)
	return a.RunWithTraceID(ctx, userMessage, traceID)
}

// runAgentWithSession 在会话语义下运行 Agent（默认沿用共享 memory manager）。
func runAgentWithSession(
	ctx context.Context,
	userMessage string,
	sessionID string,
	traceID string,
	store *ConfigStore,
	sessionStore *session.Store,
) (string, string, error) {
	return runAgentWithSessionAndMemory(ctx, userMessage, sessionID, traceID, store, sessionStore, nil)
}

// newSessionAgentExecutor 负责组装服务默认的会话执行器。
func newSessionAgentExecutor(sharedMemoryManager *memory.MemoryManager) agentExecutorFunc {
	return func(
		ctx context.Context,
		userMessage string,
		sessionID string,
		traceID string,
		store *ConfigStore,
		sessionStore *session.Store,
	) (string, string, error) {
		return runAgentWithSessionAndMemory(ctx, userMessage, sessionID, traceID, store, sessionStore, sharedMemoryManager)
	}
}

// runAgentWithSessionAndMemory 允许注入共享 memory manager，便于服务复用热态记忆。
func runAgentWithSessionAndMemory(
	ctx context.Context,
	userMessage string,
	sessionID string,
	traceID string,
	store *ConfigStore,
	sessionStore *session.Store,
	sharedMemoryManager *memory.MemoryManager,
) (string, string, error) {
	deps, err := buildAgentRuntimeDependencies(store)
	if err != nil {
		return "", "", err
	}

	executor := newAgentSessionRuntime(deps, sessionStore, sharedMemoryManager, traceID)
	return executor.run(ctx, userMessage, sessionID)
}

// newAgentSessionRuntime 绑定会话执行依赖，并在缺省时创建本地 memory manager。
func newAgentSessionRuntime(
	deps agentRuntimeDependencies,
	sessionStore *session.Store,
	sharedMemoryManager *memory.MemoryManager,
	traceID string,
) *agentSessionExecutor {
	memoryManager := sharedMemoryManager
	if memoryManager == nil {
		memoryManager = memory.NewMemoryManager(memory.MemoryConfig{
			WarmCapacity: agentWarmMemoryCapacity,
			WarmPath:     deps.cfg.MemoryWarmPath,
			ColdBaseDir:  deps.cfg.MemoryColdPath,
			SessionStore: sessionStore,
		})
	}

	return &agentSessionExecutor{
		deps:          deps,
		sessionStore:  sessionStore,
		memoryManager: memoryManager,
		traceID:       strings.TrimSpace(traceID),
	}
}

// run 负责一次“加载会话 -> 推理 -> 持久化 -> 归档”的完整回合。
func (e *agentSessionExecutor) run(ctx context.Context, userMessage string, sessionID string) (string, string, error) {
	sess, err := e.loadOrCreateSession(sessionID)
	if err != nil {
		return "", "", err
	}

	history := e.buildHistory(sess)
	e.memoryManager.SetHotContext(sess.ID, history)
	defer e.archiveSession(sess.ID)

	a := agent.NewAgentWithHistory(e.deps.client, e.deps.registry, history, e.deps.cfg.MaxTurns)
	execCtx := tools.WithSession(ctx, sess)

	response, err := a.RunWithTraceID(execCtx, userMessage, e.traceID)
	if err != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(err, &awaitingErr) {
			if saveErr := e.persistSessionMessages(sess, a.GetNewMessages()); saveErr != nil {
				return "", "", saveErr
			}
			if e.sessionStore == nil {
				return "", "", err
			}
			return "", strings.TrimSpace(sess.ID), err
		}
		return "", "", err
	}

	if saveErr := e.persistSessionMessages(sess, a.GetNewMessages()); saveErr != nil {
		return "", "", saveErr
	}
	if e.sessionStore == nil {
		return response, "", nil
	}
	return response, strings.TrimSpace(sess.ID), nil
}

// loadOrCreateSession 优先加载已有会话，失败时回退为新会话。
func (e *agentSessionExecutor) loadOrCreateSession(sessionID string) (*session.Session, error) {
	trimmedSessionID := strings.TrimSpace(sessionID)
	if e.sessionStore != nil && trimmedSessionID != "" {
		sess, err := e.sessionStore.Load(trimmedSessionID)
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			return nil, err
		}
		if err == nil {
			return sess, nil
		}
	}

	// 首次会话没有历史，按当前 system prompt 创建空会话。
	return session.NewSession(e.deps.systemPrompt), nil
}

// buildHistory 基于会话历史恢复 Agent 上下文，并注入已回答的人类反馈。
func (e *agentSessionExecutor) buildHistory(sess *session.Session) *agent.History {
	// 把上一轮已回答的 ask_human 结果注入为 tool 消息，恢复中断链路。
	injectAnsweredHumanResponses(sess)

	contextLimit := session.GetContextLimit(e.deps.cfg.Provider, e.deps.cfg.Model)
	messages := messagesWithSystemPrompt(sess.GetMessages(contextLimit), e.deps.systemPrompt)
	return agent.NewHistoryFromMessages(messages)
}

// persistSessionMessages 将本轮新增消息写回会话存储。
func (e *agentSessionExecutor) persistSessionMessages(sess *session.Session, messages []llm.Message) error {
	if e.sessionStore == nil {
		return nil
	}
	for _, msg := range messages {
		sess.AddMessage(msg)
	}
	return e.sessionStore.Save(sess)
}

// archiveSession 在回合结束后触发冷存归档；失败仅记录日志不影响主流程。
func (e *agentSessionExecutor) archiveSession(sessionID string) {
	if e.memoryManager == nil {
		return
	}
	if err := e.memoryManager.ArchiveToCold(sessionID); err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_ARCHIVE status=error session_id=%s error=%v",
			e.traceID,
			strings.TrimSpace(sessionID),
			err,
		)
	}
}

// buildAgentRuntimeDependencies 组装运行 Agent 所需的配置、模型客户端与工具注册表。
func buildAgentRuntimeDependencies(store *ConfigStore) (agentRuntimeDependencies, error) {
	var runtimeCfg runtimeConfig
	var err error
	if store == nil {
		runtimeCfg, err = runtimeConfigFromEnv()
		if err != nil {
			return agentRuntimeDependencies{}, err
		}
	} else {
		runtimeCfg = store.RuntimeConfig()
	}

	cfg, err := loadConfigWithRuntime(runtimeCfg)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}

	client := llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              cfg.Model,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	})

	executionClient := execution.NewNativeClient()
	registry := tools.NewRegistry()
	registry.Register(tools.NewScriptExecTool(executionClient))
	registry.Register(tools.NewWebSearchTool())
	registry.Register(tools.NewBrowserActionTool(executionClient))
	registry.Register(tools.NewAskHumanTool())

	promptManager, err := ctxmgr.NewPromptManager(cfg.PromptsPath)
	if err != nil {
		promptManager = ctxmgr.NewPromptManagerWithDefault()
	}
	contextBuilder := ctxmgr.NewBuilder(promptManager, registry)
	systemPrompt := contextBuilder.BuildSystemPrompt(map[string]string{
		"os_type":     runtime.GOOS,
		"tools_count": strconv.Itoa(len(registry.ToolDefs())),
		"max_turns":   strconv.Itoa(cfg.MaxTurns),
	})

	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       client,
		registry:     registry,
		systemPrompt: systemPrompt,
	}, nil
}

// injectAnsweredHumanResponses 将会话中已答复的人类问题转换为 tool result 消息。
func injectAnsweredHumanResponses(sess *session.Session) {
	if sess == nil {
		return
	}

	// PopAnsweredQuestions 只返回“尚未注入历史”的回答，避免重复消费。
	resolved := sess.PopAnsweredQuestions()
	for _, item := range resolved {
		toolCallID := strings.TrimSpace(item.Question.ToolCallID)
		if toolCallID == "" {
			continue
		}

		payload := map[string]any{
			"question_id": item.QuestionID,
			"prompt":      item.Question.Prompt,
			"answer":      item.Answer,
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			encoded = []byte(`{}`)
		}

		traceID := strings.TrimSpace(item.Question.TraceID)
		// 对齐 agent 侧 tool result envelope，保证后续轮次可无缝推理。
		sess.AddMessage(agentMessageForAskHuman(toolCallID, traceID, string(encoded)))
	}
}

// agentMessageForAskHuman 构造 ask_human 的 tool 消息，供下一轮继续推理。
func agentMessageForAskHuman(toolCallID string, traceID string, output string) llm.Message {
	return llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       agent.FormatToolResult("ask_human", traceID, output, nil),
	}
}

// messagesWithSystemPrompt 在请求构建阶段覆盖 system prompt，避免修改会话持久化内容。
func messagesWithSystemPrompt(messages []llm.Message, systemPrompt string) []llm.Message {
	out := llm.CloneMessages(messages)
	prompt := strings.TrimSpace(systemPrompt)
	if prompt == "" {
		return out
	}

	desired := llm.Message{
		Role: llm.RoleSystem,
		Text: prompt,
	}

	switch {
	case len(out) == 0:
		return []llm.Message{desired}
	case out[0].Role == llm.RoleSystem:
		out[0] = desired
		return out
	default:
		updated := make([]llm.Message, 0, len(out)+1)
		updated = append(updated, desired)
		updated = append(updated, out...)
		return updated
	}
}
