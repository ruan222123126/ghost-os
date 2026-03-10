package app

import "ghost-os/bridge/execution"

type executionClientConfig struct {
	Persistent             bool
	NativeBinaryPath       string
	NativeBinaryRoots      []string
	NativeBinaryCandidates []string
}

func newExecutionClient(cfg executionClientConfig) execution.Client {
	return execution.NewClientWithOptions(execution.ClientOptions{
		Persistent:             cfg.Persistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      cfg.NativeBinaryRoots,
		NativeBinaryCandidates: cfg.NativeBinaryCandidates,
	})
}

func executionClientConfigFromConfig(cfg Config) executionClientConfig {
	return executionClientConfig{
		Persistent:             cfg.NativePersistent,
		NativeBinaryPath:       cfg.NativeBinaryPath,
		NativeBinaryRoots:      append([]string(nil), cfg.NativeBinaryRoots...),
		NativeBinaryCandidates: append([]string(nil), cfg.NativeBinaryCandidates...),
	}
}

func executionClientConfigFromEnv() executionClientConfig {
	return executionClientConfig{
		Persistent:             nativePersistentEnabledFromEnv(),
		NativeBinaryPath:       nativeBinaryPathFromEnv(),
		NativeBinaryRoots:      nativeBinaryRootsFromEnv(),
		NativeBinaryCandidates: nativeBinaryCandidatesFromEnv(),
	}
}

func closeExecutionClient(client execution.Client) error {
	return execution.CloseClient(client)
}
