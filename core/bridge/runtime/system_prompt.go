package runtime

import (
	"fmt"
	goruntime "runtime"
	"strconv"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/session"
	"ghost-os/bridge/skills"
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
	basePrompt := renderSystemPrompt(promptManager, cfg, catalog, sess, idleTurns, systemPromptOverrides{})
	systemPrompts, err := bridgeconfig.LoadSystemPromptFiles(cfg.PromptsDir)
	if err != nil {
		return "", fmt.Errorf("load system prompts: %w", err)
	}
	if err := prepareMemoryMode(cfg); err != nil {
		return "", fmt.Errorf("prepare memory mode: %w", err)
	}
	contextSection, hasActiveContextCards, err := resolveContextSection(
		cfg,
		promptManager,
		systemPrompts.PromptLibrary,
	)
	if err != nil {
		return "", fmt.Errorf("resolve context section: %w", err)
	}
	overrides := systemPromptOverrides{
		coreJob: strings.TrimSpace(systemPrompts.CorePrompt),
	}
	if ruleOverride, found := activePromptContentForInsertPoint(
		systemPrompts.PromptLibrary,
		bridgeconfig.SystemPromptInsertPointRule,
	); found {
		overrides.rule = &ruleOverride
	}
	if hasActiveContextCards {
		overrides.contextSection = contextSection
	}
	if !overrides.hasOverrides() {
		return basePrompt, nil
	}
	return renderSystemPrompt(promptManager, cfg, catalog, sess, idleTurns, overrides), nil
}

func buildBaseSystemPrompt(
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
	overrides systemPromptOverrides,
) (string, error) {
	promptManager, err := loadPromptManager(cfg)
	if err != nil {
		return "", err
	}
	return renderSystemPrompt(promptManager, cfg, catalog, sess, idleTurns, overrides), nil
}

func renderSystemPrompt(
	promptManager *ctxmgr.PromptManager,
	cfg Config,
	catalog tools.ToolCatalog,
	sess *session.Session,
	idleTurns int,
	overrides systemPromptOverrides,
) string {
	skillDiscovery := skills.DiscoverRuntimeVisibleSkills(skillRuntimeConfig(cfg))
	vars := systemPromptVars(
		cfg,
		catalog,
		sess,
		idleTurns,
		skillDiscovery,
		resolvePromptContextSection(cfg, overrides.contextSection),
	)
	if overrides.rule != nil {
		vars["rule"] = strings.TrimSpace(*overrides.rule)
	}
	if trimmed := strings.TrimSpace(overrides.coreJob); trimmed != "" {
		vars["core_job"] = trimmed
	}
	return promptManager.Render(vars)
}

type systemPromptOverrides struct {
	rule           *string
	coreJob        string
	contextSection string
}

func (o systemPromptOverrides) hasOverrides() bool {
	return o.rule != nil ||
		o.coreJob != "" ||
		strings.TrimSpace(o.contextSection) != ""
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
	skillDiscovery skills.DiscoveryResult,
	contextSection string,
) map[string]string {
	return map[string]string{
		"os_type":            goruntime.GOOS,
		"context":            strings.TrimSpace(contextSection),
		"skills_catalog":     formatVisibleSkillsCatalogFromDiscovery(skillDiscovery),
		"dynamic_tool_state": formatDynamicToolState(sess, idleTurns),
		"dynamic_skill_context": formatDynamicSkillContextFromDiscovery(
			sess,
			idleTurns,
			skillDiscovery,
		),
		"max_turns":    strconv.Itoa(cfg.MaxTurns),
		"project_root": resolvePromptProjectRoot(cfg.ProjectRoot),
	}
}
