package config

import "ghost-os/bridge/llm"

// PublicSnapshot 返回给配置 UI/CLI 的可编辑配置视图。
func (s *store) PublicSnapshot() (Snapshot, error) {
	s.mu.RLock()
	runtime := cloneRuntimeConfig(s.runtime)
	fileCfg, _, err := s.loadStoredFileConfigLocked()
	s.mu.RUnlock()
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := snapshotFromRuntimeConfig(runtime)
	snapshot.ChatPath = publicChatPathValue(runtime, fileCfg)
	snapshot.ProjectRoot = publicProjectRootValue(fileCfg)
	return snapshot, nil
}

func publicChatPathValue(runtime runtimeConfig, fileCfg bridgeFileConfig) string {
	if chatPath := stringValue(fileCfg.ChatPath); chatPath != "" {
		return chatPath
	}
	if runtime.ChatPath == defaultChatPathForProvider(runtime.Provider) {
		return ""
	}
	return runtime.ChatPath
}

func publicProjectRootValue(fileCfg bridgeFileConfig) string {
	return stringValue(fileCfg.ProjectRoot)
}

func defaultChatPathForProvider(provider llm.Provider) string {
	switch provider.Normalized() {
	case llm.ProviderAnthropic:
		return "/v1/messages"
	case llm.ProviderCodex:
		return "/responses"
	default:
		return "/chat/completions"
	}
}
