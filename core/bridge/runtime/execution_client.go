package runtime

import "ghost-os/bridge/execution"

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

func executionClientConfigFromEnv() executionClientConfig {
	return executionClientConfig{
		Persistent:             nativePersistentEnabledFromEnv(),
		NativeBinaryPath:       nativeBinaryPathFromEnv(),
		NativeBinaryRoots:      nativeBinaryRootsFromEnv(),
		NativeBinaryCandidates: nativeBinaryCandidatesFromEnv(),
		AllowedReadPaths:       nativeAllowedReadPathsFromEnv(),
		AllowedWritePaths:      nativeAllowedWritePathsFromEnv(),
		WorkingDir:             projectRootFromEnv(),
	}
}

func closeExecutionClient(client execution.Client) error {
	return execution.CloseClient(client)
}
