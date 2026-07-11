package runtime

import (
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func newToolSelectorFromConfig(cfg Config, catalog tools.ToolCatalog) selectorEngine {
	if !cfg.ToolSelector.Enabled {
		return nil
	}
	if mode := strings.ToLower(strings.TrimSpace(cfg.ToolSelector.Mode)); mode != "" && mode != "llm" {
		return nil
	}

	client := llm.NewClientWithOptions(providerClientOptions(cfg, toolSelectorModel(cfg)))
	return NewToolSelectorForCatalog(cfg, client, catalog)
}

func toolSelectorModel(cfg Config) string {
	if model := strings.TrimSpace(cfg.ToolSelector.Model); model != "" {
		return model
	}
	if model := strings.TrimSpace(cfg.Worker.Model); model != "" {
		return model
	}
	return strings.TrimSpace(cfg.Provider.Model)
}

func buildSystemPromptForCatalog(cfg Config, catalog tools.ToolCatalog) (string, error) {
	return buildSystemPrompt(cfg, catalog)
}
