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

func newExecutionClient(cfg executionClientConfig) execution.Client {
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
	return executionClientConfig{
		Persistent:             cfg.NativePersistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      append([]string(nil), cfg.NativeBinaryRoots...),
		NativeBinaryCandidates: append([]string(nil), cfg.NativeBinaryCandidates...),
		AllowedReadPaths:       append([]string(nil), cfg.NativeAllowedReadPaths...),
		AllowedWritePaths:      append([]string(nil), cfg.NativeAllowedWritePaths...),
		WorkingDir:             cfg.ProjectRoot,
	}
}

func executionClientConfigFromEnv() (executionClientConfig, error) {
	cfg, err := bridgeconfig.LoadExecutionConfig()
	if err != nil {
		return executionClientConfig{}, err
	}
	return executionClientConfig{
		Persistent:             cfg.Persistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      append([]string(nil), cfg.NativeBinaryRoots...),
		NativeBinaryCandidates: append([]string(nil), cfg.NativeBinaryCandidates...),
		AllowedReadPaths:       append([]string(nil), cfg.AllowedReadPaths...),
		AllowedWritePaths:      append([]string(nil), cfg.AllowedWritePaths...),
		WorkingDir:             cfg.ProjectRoot,
	}, nil
}

func closeExecutionClient(client execution.Client) error {
	return execution.CloseClient(client)
}
