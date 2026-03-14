package orchestration

import (
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
)

func LoadConfig() (Config, error) {
	return bridgeconfig.Load()
}

func getenvDefault(name, fallback string) string {
	return bridgeconfig.GetenvDefault(name, fallback)
}

func parseStringCSV(raw string) []string {
	return bridgeconfig.ParseStringCSV(raw)
}

func runtimeConfigFromEnv() (runtimeConfig, error) {
	return bridgeconfig.RuntimeConfigFromEnv()
}

func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	return bridgeconfig.LoadWithRuntime(runtime)
}

func providerClientOptions(cfg Config, model string) llm.ClientOptions {
	return bridgeconfig.ProviderClientOptions(cfg, model)
}

func valueOrEnv(raw *string, envName, fallback string) string {
	return bridgeconfig.ValueOrEnv(raw, envName, fallback)
}

func sessionsPathFromEnv() string {
	return bridgeconfig.SessionsPathFromEnv()
}

func rssFeedsPathFromEnv() string {
	return bridgeconfig.RSSFeedsPathFromEnv()
}

func rssInboxPathFromEnv() string {
	return bridgeconfig.RSSInboxPathFromEnv()
}

func rssBriefingsPathFromEnv() string {
	return bridgeconfig.RSSBriefingsPathFromEnv()
}

func rssReportsPathFromEnv() string {
	return bridgeconfig.RSSReportsPathFromEnv()
}

func rssPollEnabledFromEnv() bool {
	return bridgeconfig.RSSPollEnabledFromEnv()
}

func rssPollIntervalFromEnv() time.Duration {
	return bridgeconfig.RSSPollIntervalFromEnv()
}

func rssPollMaxItemsPerFeedFromEnv() int {
	return bridgeconfig.RSSPollMaxItemsPerFeedFromEnv()
}

func rssAIBatchSizeFromEnv() int {
	return bridgeconfig.RSSAIBatchSizeFromEnv()
}

func rssBriefingEnabledFromEnv() bool {
	return bridgeconfig.RSSBriefingEnabledFromEnv()
}

func rssBriefingIntervalFromEnv() time.Duration {
	return bridgeconfig.RSSBriefingIntervalFromEnv()
}

func corsOriginsOrEnv(raw []string) []string {
	return bridgeconfig.CORSOriginsOrEnv(raw)
}

func webSearchTavilyAPIKeyFromEnv() string {
	return bridgeconfig.WebSearchTavilyAPIKeyFromEnv()
}

func tasksPathFromEnv() string {
	return bridgeconfig.TasksPathFromEnv()
}

func nativePersistentEnabledFromEnv() bool {
	return bridgeconfig.NativePersistentEnabledFromEnv()
}

func nativeBinaryPathFromEnv() string {
	return bridgeconfig.NativeBinaryPathFromEnv()
}

func nativeBinaryRootsFromEnv() []string {
	return bridgeconfig.NativeBinaryRootsFromEnv()
}

func nativeBinaryCandidatesFromEnv() []string {
	return bridgeconfig.NativeBinaryCandidatesFromEnv()
}

func nativeAllowedReadPathsFromEnv() []string {
	return bridgeconfig.NativeAllowedReadPathsFromEnv()
}

func nativeAllowedWritePathsFromEnv() []string {
	return bridgeconfig.NativeAllowedWritePathsFromEnv()
}

func projectRootFromEnv() string {
	return bridgeconfig.ProjectRootFromEnv()
}

func defaultBaseURLForProvider(provider llm.Provider) string {
	return bridgeconfig.DefaultBaseURLForProvider(provider)
}
