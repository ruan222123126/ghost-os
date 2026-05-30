package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
)

const (
	migratePromptsDirSubcommand     = "prompts-dir"
	migrateToolPromptsSubcommand    = "tool-prompts"
	migrateSystemPromptsSubcommand  = "system-prompts"
	migrateSessionsSubcommand       = "sessions"
	migrateOrchestrationsSubcommand = "orchestrations"
)

func runMigrateCommand(_ context.Context, args []string) (string, error) {
	subcommand := strings.TrimSpace(args[0])
	traceID := buildMigrateTraceID(subcommand)

	switch subcommand {
	case migratePromptsDirSubcommand:
		report, err := bridgeconfig.MigrateLegacyPromptsDir()
		if err != nil {
			return "", err
		}
		return formatPromptsDirMigration(traceID, report), nil
	case migrateToolPromptsSubcommand:
		report, err := bridgeconfig.MigrateLegacyToolPrompts()
		if err != nil {
			return "", err
		}
		return formatToolPromptsMigration(traceID, report), nil
	case migrateSystemPromptsSubcommand:
		report, err := bridgeconfig.MigrateLegacySystemPrompts()
		if err != nil {
			return "", err
		}
		return formatSystemPromptsMigration(traceID, report), nil
	case migrateSessionsSubcommand:
		cfg, err := bridgeconfig.LoadServerConfig()
		if err != nil {
			return "", err
		}
		store, err := session.NewStore(cfg.SessionsPath)
		if err != nil {
			return "", err
		}
		defer store.Close()
		migrated, err := store.MigrateLegacySessions()
		if err != nil {
			return "", err
		}
		return formatSessionMigration(traceID, cfg.SessionsPath, migrated), nil
	case migrateOrchestrationsSubcommand:
		taskCfg, err := bridgeconfig.LoadTaskConfig()
		if err != nil {
			return "", err
		}
		report, err := bridgeorchestration.MigrateLegacyOrchestrations(taskCfg.TasksPath)
		if err != nil {
			return "", err
		}
		return formatOrchestrationMigration(traceID, report), nil
	default:
		return "", newUsageError(fmt.Sprintf("unknown migrate subcommand %q", subcommand))
	}
}

func buildMigrateTraceID(name string) string {
	return fmt.Sprintf("migrate-%s-%s", strings.TrimSpace(name), time.Now().UTC().Format("20060102T150405Z"))
}

func formatPromptsDirMigration(traceID string, report bridgeconfig.PromptsDirMigrationReport) string {
	lines := []string{
		"trace_id=" + traceID,
		"migration=prompts-dir",
		"prompts_dir=" + report.PromptsDir,
		"canonical_dir=" + report.CanonicalDir,
		fmt.Sprintf("merged_file_count=%d", len(report.MergedFiles)),
		fmt.Sprintf("config_updated=%t", report.ConfigUpdated),
	}
	if report.BackupDir != "" {
		lines = append(lines, "backup_dir="+report.BackupDir)
	}
	if len(report.MergedFiles) > 0 {
		lines = append(lines, "merged_files="+strings.Join(report.MergedFiles, ","))
	}
	return strings.Join(lines, "\n")
}

func formatToolPromptsMigration(traceID string, report bridgeconfig.ToolPromptsMigrationReport) string {
	lines := []string{
		"trace_id=" + traceID,
		"migration=tool-prompts",
		"prompts_dir=" + report.PromptsDir,
		fmt.Sprintf("migrated_tool_count=%d", len(report.MigratedTools)),
		fmt.Sprintf("skipped_tool_count=%d", len(report.SkippedTools)),
		fmt.Sprintf("config_updated=%t", report.ConfigUpdated),
	}
	if len(report.MigratedTools) > 0 {
		lines = append(lines, "migrated_tools="+strings.Join(report.MigratedTools, ","))
	}
	if len(report.SkippedTools) > 0 {
		lines = append(lines, "skipped_tools="+strings.Join(report.SkippedTools, ","))
	}
	return strings.Join(lines, "\n")
}

func formatSystemPromptsMigration(traceID string, report bridgeconfig.SystemPromptsMigrationReport) string {
	lines := []string{
		"trace_id=" + traceID,
		"migration=system-prompts",
		"prompts_dir=" + report.PromptsDir,
		fmt.Sprintf("archived_file_count=%d", len(report.ArchivedFiles)),
		fmt.Sprintf("generated_prompt_library=%t", report.GeneratedPromptLib),
		fmt.Sprintf("migrated_memory_card_count=%d", len(report.MigratedMemoryCards)),
		fmt.Sprintf("cleared_preset_memory_count=%d", len(report.ClearedPresetMemory)),
	}
	if len(report.ArchivedFiles) > 0 {
		lines = append(lines, "archived_files="+strings.Join(report.ArchivedFiles, ","))
	}
	if len(report.MigratedMemoryCards) > 0 {
		lines = append(lines, "migrated_memory_cards="+strings.Join(report.MigratedMemoryCards, ","))
	}
	if len(report.ClearedPresetMemory) > 0 {
		lines = append(lines, "cleared_preset_memory="+strings.Join(report.ClearedPresetMemory, ","))
	}
	return strings.Join(lines, "\n")
}

func formatSessionMigration(traceID string, sessionsPath string, migrated []string) string {
	lines := []string{
		"trace_id=" + traceID,
		"migration=sessions",
		"sessions_path=" + sessionsPath,
		fmt.Sprintf("migrated_session_count=%d", len(migrated)),
	}
	if len(migrated) > 0 {
		lines = append(lines, "migrated_sessions="+strings.Join(migrated, ","))
	}
	return strings.Join(lines, "\n")
}

func formatOrchestrationMigration(
	traceID string,
	report bridgeorchestration.LegacyOrchestrationMigrationReport,
) string {
	lines := []string{
		"trace_id=" + traceID,
		"migration=orchestrations",
		"tasks_path=" + report.TasksPath,
		fmt.Sprintf("migrated_task_count=%d", len(report.MigratedTaskIDs)),
	}
	if len(report.MigratedTaskIDs) > 0 {
		lines = append(lines, "migrated_tasks="+strings.Join(report.MigratedTaskIDs, ","))
	}
	return strings.Join(lines, "\n")
}
