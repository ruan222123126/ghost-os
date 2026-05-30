package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	legacyPromptsDirMigrateCommand    = "bin/ghost-bridge migrate prompts-dir"
	legacyToolPromptsMigrateCommand   = "bin/ghost-bridge migrate tool-prompts"
	legacySystemPromptsMigrateCommand = "bin/ghost-bridge migrate system-prompts"
)

type LegacyFinding struct {
	Code           string
	Message        string
	MigrateCommand string
}

type PromptsDirMigrationReport struct {
	PromptsDir    string
	CanonicalDir  string
	BackupDir     string
	MergedFiles   []string
	ConfigUpdated bool
}

type ToolPromptsMigrationReport struct {
	PromptsDir    string
	MigratedTools []string
	SkippedTools  []string
	ConfigUpdated bool
}

// DetectLegacyPromptState 扫描 prompts/config 相关 legacy 形态，供 serve preflight 显式 fail-fast。
func DetectLegacyPromptState() ([]LegacyFinding, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return nil, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return nil, err
	}

	findings := make([]LegacyFinding, 0, 3)
	if hasLegacyPromptsDirState(promptsDir) {
		findings = append(findings, LegacyFinding{
			Code:           "legacy_prompts_dir",
			Message:        fmt.Sprintf("legacy prompts roots detected for %s", promptsDir),
			MigrateCommand: legacyPromptsDirMigrateCommand,
		})
	}
	if len(fileCfg.ToolPromptOverrides) > 0 {
		findings = append(findings, LegacyFinding{
			Code:           "legacy_tool_prompt_overrides",
			Message:        "config.toml still contains tool_prompt_overrides",
			MigrateCommand: legacyToolPromptsMigrateCommand,
		})
	}
	if hasLegacySystemPromptState(promptsDir) {
		findings = append(findings, LegacyFinding{
			Code:           "legacy_system_prompts",
			Message:        fmt.Sprintf("legacy system prompt state detected under %s", promptsDir),
			MigrateCommand: legacySystemPromptsMigrateCommand,
		})
	}
	return findings, nil
}

// MigrateLegacyPromptsDir 合并 .ghost/.ghost-os prompts 根目录，并统一 canonical 到 ~/.ghost-os/prompts。
func MigrateLegacyPromptsDir() (PromptsDirMigrationReport, error) {
	fileCfg, configPath, err := loadBridgeFileConfig()
	if err != nil {
		return PromptsDirMigrationReport{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return PromptsDirMigrationReport{}, err
	}
	canonicalDir, legacyDir, ok := promptMirrorPair(promptsDir)
	if !ok {
		return PromptsDirMigrationReport{
			PromptsDir:   promptsDir,
			CanonicalDir: promptsDir,
		}, nil
	}

	report := PromptsDirMigrationReport{
		PromptsDir:   promptsDir,
		CanonicalDir: canonicalDir,
	}
	if err := os.MkdirAll(canonicalDir, toolPromptDirPerm); err != nil {
		return PromptsDirMigrationReport{}, fmt.Errorf("create canonical prompts dir %s: %w", canonicalDir, err)
	}

	mergedFiles, err := mergePromptManagedFiles(canonicalDir, legacyDir)
	if err != nil {
		return PromptsDirMigrationReport{}, err
	}
	report.MergedFiles = mergedFiles

	if _, err := ensureToolPromptRoots(canonicalDir, toolBasePrompts()); err != nil {
		return PromptsDirMigrationReport{}, err
	}
	if _, err := ensureSystemPromptRoots(canonicalDir, defaultSystemPromptFiles()); err != nil {
		return PromptsDirMigrationReport{}, err
	}
	if _, err := ensurePresetRoot(canonicalDir); err != nil {
		return PromptsDirMigrationReport{}, err
	}

	if resolvedStoredPromptsDir(fileCfg) == legacyDir {
		fileCfg.PromptsDir = stringPointer(canonicalDir)
		if err := writeBridgeFileConfig(configPath, fileCfg); err != nil {
			return PromptsDirMigrationReport{}, err
		}
		report.ConfigUpdated = true
	}

	if legacyDir == canonicalDir || !pathExists(legacyDir) {
		return report, nil
	}
	backupDir := fmt.Sprintf("%s.legacy.bak.%d", legacyDir, time.Now().UTC().Unix())
	if err := os.Rename(legacyDir, backupDir); err != nil {
		return PromptsDirMigrationReport{}, fmt.Errorf("backup legacy prompts dir %s: %w", legacyDir, err)
	}
	report.BackupDir = backupDir
	return report, nil
}

// MigrateLegacyToolPrompts 将 config.toml 中的 tool_prompt_overrides 显式迁移到 prompts 文件。
func MigrateLegacyToolPrompts() (ToolPromptsMigrationReport, error) {
	fileCfg, configPath, err := loadBridgeFileConfig()
	if err != nil {
		return ToolPromptsMigrationReport{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return ToolPromptsMigrationReport{}, err
	}
	report := ToolPromptsMigrationReport{PromptsDir: promptsDir}
	if len(fileCfg.ToolPromptOverrides) == 0 {
		return report, nil
	}

	current, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		return ToolPromptsMigrationReport{}, err
	}
	defaults := toolBasePrompts()
	for rawName, rawPrompt := range fileCfg.ToolPromptOverrides {
		name := strings.TrimSpace(rawName)
		prompt := strings.TrimSpace(rawPrompt)
		if name == "" || prompt == "" {
			continue
		}
		currentPrompt := strings.TrimSpace(current[name])
		if currentPrompt != "" && currentPrompt != strings.TrimSpace(defaults[name]) {
			report.SkippedTools = append(report.SkippedTools, name)
			continue
		}
		if err := writeToolPromptOverrideToFile(promptsDir, name, prompt); err != nil {
			return ToolPromptsMigrationReport{}, fmt.Errorf("migrate tool prompt override %s: %w", name, err)
		}
		report.MigratedTools = append(report.MigratedTools, name)
	}

	fileCfg.ToolPromptOverrides = nil
	if err := writeBridgeFileConfig(configPath, fileCfg); err != nil {
		return ToolPromptsMigrationReport{}, err
	}
	report.ConfigUpdated = true
	return report, nil
}
