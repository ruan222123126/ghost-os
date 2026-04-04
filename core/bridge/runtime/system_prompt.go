package runtime

import (
	"fmt"
	goruntime "runtime"
	"strconv"

	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func buildSystemPrompt(cfg Config, catalog tools.ToolCatalog) (string, error) {
	return buildSystemPromptForSession(cfg, catalog, nil, cfg.ToolSearch.IdleTurns)
}

func buildSystemPromptForSession(
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
) (string, error) {
	promptManager, err := loadPromptManager(cfg)
	if err != nil {
		return "", err
	}
	return promptManager.Render(systemPromptVars(cfg, catalog, sess, idleTurns)), nil
}

func loadPromptManager(cfg Config) (*ctxmgr.PromptManager, error) {
	promptManager, err := ctxmgr.NewPromptManagerWithOptions(ctxmgr.PromptLoadOptions{
		ConfigPath:             cfg.PromptsPath,
		CoreDir:                cfg.PromptsDir,
		CoreFiles:              cfg.PromptsCoreFiles,
		RuntimeConstraintFiles: cfg.PromptsRuntimeConstraintFiles,
		ResponseRuleFiles:      cfg.PromptsResponseRuleFiles,
	})
	if err != nil {
		return nil, fmt.Errorf("load prompt manager: %w", err)
	}
	return promptManager, nil
}

func systemPromptVars(
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
) map[string]string {
	return map[string]string{
		"os_type":            goruntime.GOOS,
		"tool_guidance":      tools.FormatPromptGuidanceForCatalog(catalog),
		"dynamic_tool_state": formatDynamicToolState(sess, idleTurns),
		"max_turns":          strconv.Itoa(cfg.MaxTurns),
		"project_root":       resolvePromptProjectRoot(cfg.ProjectRoot),
	}
}
