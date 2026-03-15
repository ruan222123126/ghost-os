package transport

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
)

const (
	defaultBaseURL = bridgeconfig.DefaultBaseURL
	defaultModel   = bridgeconfig.DefaultModel
)

type ConfigStore = bridgeorchestration.ConfigStore
type Config = bridgeorchestration.Config
type ProviderConfig = bridgeorchestration.ProviderConfig
type WorkerConfig = bridgeorchestration.WorkerConfig
type bridgeFileConfig = bridgeconfig.FileConfig
type providerFileConfig = bridgeconfig.ProviderFileConfig

func NewConfigStoreFromEnv() (*ConfigStore, error) {
	return bridgeorchestration.NewConfigStoreFromEnv()
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	return bridgeconfig.LoadBridgeFileConfig()
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return bridgeconfig.WriteBridgeFileConfig(path, cfg)
}

func sessionsPathFromEnv() string {
	return bridgeconfig.SessionsPathFromEnv()
}

func configPathFromEnv() string {
	return bridgeconfig.ConfigPathFromEnv()
}

func valueOrEnv(raw *string, envName, fallback string) string {
	return bridgeconfig.ValueOrEnv(raw, envName, fallback)
}

func getenvDefault(name, fallback string) string {
	return bridgeconfig.GetenvDefault(name, fallback)
}

func corsOriginsOrEnv(raw []string) []string {
	return bridgeconfig.CORSOriginsOrEnv(raw)
}

func optionalStringPointer(raw string) *string {
	return bridgeconfig.OptionalStringPointer(raw)
}
