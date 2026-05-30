package config

import (
	"fmt"
	"strings"
)

func resolveSessionsPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return valueOrEnvWithEnv(fileCfg.SessionsPath, env, "GHOST_SESSIONS_PATH", defaultSessionsPath)
}

func resolveWebSearchTavilyAPIKey(fileCfg bridgeFileConfig, env envSnapshot) string {
	if fileCfg.WebSearchTavilyAPIKey != nil {
		return strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	return env.firstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
}

func resolveTasksPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return valueOrEnvWithEnv(fileCfg.TasksPath, env, "GHOST_TASKS_PATH", defaultTasksPath)
}

func resolveScriptExecSandboxMemoryMB(fileCfg bridgeFileConfig, env envSnapshot) (int, error) {
	const fieldName = "script_exec_sandbox_memory_mb"
	const envName = "GHOST_SCRIPT_EXEC_SANDBOX_MEMORY_MB"

	if fileCfg.ScriptExecSandboxMemoryMB != nil {
		value := *fileCfg.ScriptExecSandboxMemoryMB
		if value <= 0 {
			return 0, invalidScriptExecSandboxMemoryMB(fieldName, value)
		}
		if value > maxScriptExecSandboxMemoryMB {
			return 0, invalidScriptExecSandboxMemoryMB(fieldName, value)
		}
		return value, nil
	}

	value, err := parsePositiveIntValue(env.value(envName), envName, defaultScriptExecSandboxMemoryMB)
	if err != nil {
		return 0, err
	}
	if value > maxScriptExecSandboxMemoryMB {
		return 0, invalidScriptExecSandboxMemoryMB(envName, value)
	}
	return value, nil
}

func resolveNativeBinaryPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	if override := env.firstNonEmpty("GHOST_NATIVE_BINARY_PATH_OVERRIDE", "GHOST_NATIVE_BIN_OVERRIDE"); override != "" {
		return override
	}
	if fileCfg.NativeBinaryPath != nil {
		return strings.TrimSpace(*fileCfg.NativeBinaryPath)
	}
	return env.firstNonEmpty("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
}

func resolveNativeBinaryRoots(fileCfg bridgeFileConfig, env envSnapshot) []string {
	if fileCfg.NativeBinaryRoots != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryRoots)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_ROOTS", "")))
}

func resolveNativeBinaryCandidates(fileCfg bridgeFileConfig, env envSnapshot) []string {
	if fileCfg.NativeBinaryCandidates != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryCandidates)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_CANDIDATES", "")))
}

func resolveCodexCLIPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	if override := env.value("GHOST_CODEX_CLI_PATH"); override != "" {
		return strings.TrimSpace(override)
	}
	if fileCfg.CodexCLIPath != nil {
		return strings.TrimSpace(*fileCfg.CodexCLIPath)
	}
	return ""
}

func resolveNodeBinPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	if override := env.value("GHOST_NODE_BIN_PATH"); override != "" {
		return strings.TrimSpace(override)
	}
	if fileCfg.NodeBinPath != nil {
		return strings.TrimSpace(*fileCfg.NodeBinPath)
	}
	return ""
}

func resolveNativeAllowedReadPaths(fileCfg bridgeFileConfig, env envSnapshot) []string {
	if fileCfg.NativeAllowedReadPaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedReadPaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
}

func resolveNativeAllowedWritePaths(fileCfg bridgeFileConfig, env envSnapshot) []string {
	if fileCfg.NativeAllowedWritePaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedWritePaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
}

func resolveProjectRoot(fileCfg bridgeFileConfig, env envSnapshot) string {
	if fileCfg.ProjectRoot != nil {
		return strings.TrimSpace(*fileCfg.ProjectRoot)
	}
	return env.value("GHOST_PROJECT_ROOT")
}

func invalidScriptExecSandboxMemoryMB(name string, value int) error {
	return fmt.Errorf(
		"invalid %s: must be between 1 and %d, got %d",
		strings.TrimSpace(name),
		maxScriptExecSandboxMemoryMB,
		value,
	)
}
