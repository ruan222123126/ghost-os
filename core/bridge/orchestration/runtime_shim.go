package orchestration

import (
	"context"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type toolSelectorCompleter interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
	systemPrompt string
	memoryRecall memoryaug.RecallService
	memoryLearn  memoryaug.LearningService
	cleanup      func()
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

type AgentRuntimeFactory interface {
	Build(store *ConfigStore) (agentRuntimeDependencies, error)
}

type runtimeFactoryAdapter struct {
	inner bridgeruntime.AgentRuntimeFactory
}

func (f runtimeFactoryAdapter) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	var innerStore *bridgeruntime.ConfigStore
	if store != nil {
		innerStore = store.Inner()
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
		memoryRecall: deps.MemoryRecall(),
		memoryLearn:  deps.MemoryLearning(),
		cleanup:      deps.Close,
	}, nil
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return runtimeFactoryAdapter{inner: bridgeruntime.NewAgentRuntimeFactory()}
}

func newAgentRuntimeFactoryWithTaskManager(taskManager tools.TaskManager) AgentRuntimeFactory {
	return runtimeFactoryAdapter{inner: bridgeruntime.NewAgentRuntimeFactoryWithTaskManager(taskManager)}
}

type selectorEngine = bridgeruntime.SelectorEngine
type ToolSelectorResult = bridgeruntime.ToolSelectorResult
type ToolSelector = bridgeruntime.ToolSelector

type toolSelectionPolicy struct {
	inner bridgeruntime.SelectionPolicy
}

func newToolSelectionPolicy(cfg Config) toolSelectionPolicy {
	return toolSelectionPolicy{inner: bridgeruntime.NewToolSelectionPolicy(cfg)}
}

func (p toolSelectionPolicy) scopeCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	return p.inner.ScopeCatalog(catalog)
}

func (p toolSelectionPolicy) allowlistScope(available []string) []string {
	return p.inner.AllowlistScope(available)
}

func (p toolSelectionPolicy) apply(available []string, selected []string) []string {
	return p.inner.Apply(available, selected)
}

func NewToolSelector(cfg Config, worker toolSelectorCompleter) *ToolSelector {
	return bridgeruntime.NewToolSelector(cfg, worker)
}

func NewToolSelectorForCatalog(cfg Config, worker toolSelectorCompleter, catalog tools.ToolCatalog) *ToolSelector {
	return bridgeruntime.NewToolSelectorForCatalog(cfg, worker, catalog)
}

func newToolSelectorFromConfig(cfg Config, catalog tools.ToolCatalog) selectorEngine {
	return bridgeruntime.NewSelectorFromConfig(cfg, catalog)
}

func buildSystemPromptForCatalog(cfg Config, catalog tools.ToolCatalog) (string, error) {
	return bridgeruntime.BuildSystemPromptForCatalog(cfg, catalog)
}

func newSessionTurnCatalog(catalog tools.ToolCatalog, static []string, sess *session.Session, idleTurns int, selector bool) tools.ToolCatalog {
	return bridgeruntime.NewSessionTurnCatalog(catalog, static, sess, idleTurns, selector)
}
