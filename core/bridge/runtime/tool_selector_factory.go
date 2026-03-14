package runtime

import (
	"runtime"
	"strconv"
	"strings"

	ctxmgr "ghost-os/bridge/context"
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
	promptManager, err := ctxmgr.NewPromptManagerWithOptions(ctxmgr.PromptLoadOptions{
		ConfigPath: cfg.PromptsPath,
		CoreDir:    cfg.PromptsDir,
		CoreFiles:  cfg.PromptsCoreFiles,
	})
	if err != nil {
		if len(cfg.PromptsCoreFiles) == 0 {
			promptManager = ctxmgr.NewPromptManagerWithDefault()
		} else {
			return "", err
		}
	}

	contextBuilder := ctxmgr.NewBuilder(promptManager, catalog)
	return contextBuilder.BuildSystemPrompt(map[string]string{
		"os_type":     runtime.GOOS,
		"tools_count": strconv.Itoa(len(catalog.ToolDefs())),
		"max_turns":   strconv.Itoa(cfg.MaxTurns),
	}), nil
}
