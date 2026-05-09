package runtime

import (
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type Config = bridgeconfig.Config
type ToolSelectorConfig = bridgeconfig.ToolSelectorConfig
type ToolSearchConfig = bridgeconfig.ToolSearchConfig
type Dependencies = agentRuntimeDependencies
type SelectorEngine = selectorEngine

type SelectionPolicy struct {
	toolSelectionPolicy
}

type ConfigStore struct {
	inner bridgeconfig.Store
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

func WrapConfigStore(store bridgeconfig.Store) *ConfigStore {
	if store == nil {
		return nil
	}
	return &ConfigStore{inner: store}
}

func (s *ConfigStore) Config() (Config, error) {
	if s == nil || s.inner == nil {
		return bridgeconfig.Load()
	}
	return s.inner.Config()
}

func (s *ConfigStore) SetProjectRoot(path string) error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.SetProjectRoot(path)
}

func NewAgentRuntimeFactory() AgentRuntimeFactory {
	return newAgentRuntimeFactory()
}

func NewToolSelectionPolicy(cfg Config) SelectionPolicy {
	return SelectionPolicy{toolSelectionPolicy: newToolSelectionPolicy(cfg)}
}

func (p SelectionPolicy) ResidentCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	return p.toolSelectionPolicy.residentCatalog(catalog)
}

func (p SelectionPolicy) SelectorCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	return p.toolSelectionPolicy.selectorCatalog(catalog)
}

func (p SelectionPolicy) ResidentScope(available []string) []string {
	return p.toolSelectionPolicy.residentScope(available)
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

func BuildSystemPromptForSessionWithFiles(
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
	files bridgeconfig.SystemPromptFiles,
) (string, error) {
	return buildSystemPromptForSessionWithFiles(cfg, catalog, sess, idleTurns, files)
}

func VisibleSkillNames(cfg Config) []string {
	return visibleSkillNames(cfg)
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
		Provider:                   cfg.Provider.Type,
		BaseURL:                    cfg.Provider.BaseURL,
		APIKey:                     cfg.Provider.APIKey,
		Model:                      resolvedModel,
		ChatPath:                   cfg.ChatPath,
		Headers:                    cfg.Provider.Headers,
		AnthropicVersion:           cfg.Provider.AnthropicVersion,
		AnthropicMaxTokens:         cfg.Provider.AnthropicMaxTokens,
		CodexStatelessRetryEnabled: cfg.CodexStatelessRetryEnabled,
		ResponseOptions:            llm.CloneResponseOptions(cfg.ResponseOptions),
	}
}

func NewExecutionClientFromEnv() (execution.Client, error) {
	cfg, err := executionClientConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return newExecutionClient(cfg), nil
}

func CloseExecutionClient(client execution.Client) error {
	return closeExecutionClient(client)
}
