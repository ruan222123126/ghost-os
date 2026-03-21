package runtime

import (
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type Config = bridgeconfig.Config
type ConfigStore = bridgeconfig.Store
type ToolSelectorConfig = bridgeconfig.ToolSelectorConfig
type ToolSearchConfig = bridgeconfig.ToolSearchConfig
type Dependencies = agentRuntimeDependencies
type SelectorEngine = selectorEngine

type SelectionPolicy struct {
	toolSelectionPolicy
}

func (d agentRuntimeDependencies) Config() Config {
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

func (d agentRuntimeDependencies) MemoryRecall() memoryaug.RecallService {
	return d.memoryRecall
}

func (d agentRuntimeDependencies) MemoryLearning() memoryaug.LearningService {
	return d.memoryLearn
}

func NewAgentRuntimeFactory() AgentRuntimeFactory {
	return newAgentRuntimeFactory()
}

func NewAgentRuntimeFactoryWithTaskManager(taskManager tools.TaskManager) AgentRuntimeFactory {
	return newAgentRuntimeFactoryWithTaskManager(taskManager)
}

func NewToolSelectionPolicy(cfg Config) SelectionPolicy {
	return SelectionPolicy{toolSelectionPolicy: newToolSelectionPolicy(cfg)}
}

func (p SelectionPolicy) ScopeCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	return p.toolSelectionPolicy.scopeCatalog(catalog)
}

func (p SelectionPolicy) AllowlistScope(available []string) []string {
	return p.toolSelectionPolicy.allowlistScope(available)
}

func (p SelectionPolicy) Apply(available []string, selected []string) []string {
	return p.toolSelectionPolicy.apply(available, selected)
}

func NewSelectorFromConfig(cfg Config, catalog tools.ToolCatalog) SelectorEngine {
	return newToolSelectorFromConfig(cfg, catalog)
}

func BuildSystemPromptForCatalog(cfg Config, catalog tools.ToolCatalog) (string, error) {
	return buildSystemPromptForCatalog(cfg, catalog)
}

func BuildSystemPromptForSession(
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
) (string, error) {
	return buildSystemPromptForSession(cfg, catalog, sess, idleTurns)
}

func NewSessionTurnCatalog(catalog tools.ToolCatalog, static []string, sess *session.Session, idleTurns int, selector bool) tools.ToolCatalog {
	return newSessionTurnCatalog(catalog, static, sess, idleTurns, selector)
}

func providerClientOptions(cfg Config, model string) llm.ClientOptions {
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = strings.TrimSpace(cfg.Provider.Model)
	}
	return llm.ClientOptions{
		Provider:           cfg.Provider.Type,
		BaseURL:            cfg.Provider.BaseURL,
		APIKey:             cfg.Provider.APIKey,
		Model:              resolvedModel,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.Provider.Headers,
		AnthropicVersion:   cfg.Provider.AnthropicVersion,
		AnthropicMaxTokens: cfg.Provider.AnthropicMaxTokens,
	}
}

func NewExecutionClientFromEnv() execution.Client {
	return newExecutionClient(executionClientConfigFromEnv())
}

func CloseExecutionClient(client execution.Client) error {
	return closeExecutionClient(client)
}
