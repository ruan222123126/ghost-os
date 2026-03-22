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

func NewConfigStoreFromEnv() (*ConfigStore, error) {
	return bridgeorchestration.NewConfigStoreFromEnv()
}
