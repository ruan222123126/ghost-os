package runtime

import (
	"log"
	"os"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/execution"
)

type executionClientConfig struct {
	Persistent             bool
	NativeBinaryPath       string
	NativeBinaryRoots      []string
	NativeBinaryCandidates []string
	AllowedReadPaths       []string
	AllowedWritePaths      []string
	WorkingDir             string
}

func newExecutionClient(cfg executionClientConfig) Client {
	log.Printf(
		"runtime checkpoint component=execution persistent=%t native_binary_path=%s session_type=%s wayland_display=%s display=%s",
		cfg.Persistent,
		executionRuntimeValue(cfg.NativeBinaryPath, "<auto>"),
		executionRuntimeValue(os.Getenv("XDG_SESSION_TYPE"), "<unset>"),
		executionRuntimeValue(os.Getenv("WAYLAND_DISPLAY"), "<unset>"),
		executionRuntimeValue(os.Getenv("DISPLAY"), "<unset>"),
	)
	return execution.NewClientWithOptions(execution.ClientOptions{
		Persistent:             cfg.Persistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      cfg.NativeBinaryRoots,
		NativeBinaryCandidates: cfg.NativeBinaryCandidates,
		AllowedReadPaths:       cfg.AllowedReadPaths,
		AllowedWritePaths:      cfg.AllowedWritePaths,
		WorkingDir:             cfg.WorkingDir,
	})
}

func executionRuntimeValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func executionClientConfigFromConfig(cfg Config) executionClientConfig {
	allowedReadPaths, allowedWritePaths := defaultExecutionAllowedPaths(
		cfg.NativeAllowedReadPaths,
		cfg.NativeAllowedWritePaths,
		cfg.ProjectRoot,
	)
	return executionClientConfig{
		Persistent:             cfg.NativePersistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      append([]string(nil), cfg.NativeBinaryRoots...),
		NativeBinaryCandidates: append([]string(nil), cfg.NativeBinaryCandidates...),
		AllowedReadPaths:       allowedReadPaths,
		AllowedWritePaths:      allowedWritePaths,
		WorkingDir:             cfg.ProjectRoot,
	}
}

func interactionExecutionClientConfigFromConfig(cfg Config) executionClientConfig {
	config := executionClientConfigFromConfig(cfg)
	config.WorkingDir = ""
	return config
}

func executionClientConfigFromEnv() (executionClientConfig, error) {
	cfg, err := bridgeconfig.LoadExecutionConfig()
	if err != nil {
		return executionClientConfig{}, err
	}
	allowedReadPaths, allowedWritePaths := defaultExecutionAllowedPaths(
		cfg.AllowedReadPaths,
		cfg.AllowedWritePaths,
		cfg.ProjectRoot,
	)
	return executionClientConfig{
		Persistent:             cfg.Persistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      append([]string(nil), cfg.NativeBinaryRoots...),
		NativeBinaryCandidates: append([]string(nil), cfg.NativeBinaryCandidates...),
		AllowedReadPaths:       allowedReadPaths,
		AllowedWritePaths:      allowedWritePaths,
		WorkingDir:             cfg.ProjectRoot,
	}, nil
}

func defaultExecutionAllowedPaths(
	readPaths []string,
	writePaths []string,
	projectRoot string,
) ([]string, []string) {
	resolvedRead := append([]string(nil), readPaths...)
	resolvedWrite := append([]string(nil), writePaths...)
	trimmedRoot := strings.TrimSpace(projectRoot)
	if trimmedRoot == "" {
		return resolvedRead, resolvedWrite
	}
	if len(resolvedRead) == 0 {
		resolvedRead = []string{trimmedRoot}
	}
	if len(resolvedWrite) == 0 {
		resolvedWrite = []string{trimmedRoot}
	}
	return resolvedRead, resolvedWrite
}

func closeExecutionClient(client Client) error {
	return execution.CloseClient(client)
}
