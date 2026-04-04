package config

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/llm"
)

const responseMetadataEnvPrefix = "GHOST_RESPONSE_METADATA_"

func responseOptionsFromEnv(env envSnapshot) (llm.ResponseOptions, error) {
	store, err := parseOptionalBoolValue(env.value("GHOST_RESPONSE_STORE"), "GHOST_RESPONSE_STORE")
	if err != nil {
		return llm.ResponseOptions{}, err
	}
	metadata, err := responseMetadataFromEnv(env)
	if err != nil {
		return llm.ResponseOptions{}, err
	}
	return llm.ResponseOptions{
		PromptCacheKey:       env.value("GHOST_PROMPT_CACHE_KEY"),
		PromptCacheRetention: env.value("GHOST_PROMPT_CACHE_RETENTION"),
		SafetyIdentifier:     env.value("GHOST_SAFETY_IDENTIFIER"),
		Metadata:             metadata,
		Store:                store,
	}, nil
}

func fileResponseOptions(fileCfg bridgeFileConfig, fallback llm.ResponseOptions) (llm.ResponseOptions, error) {
	out := llm.CloneResponseOptions(fallback)
	if fileCfg.ResponsePromptCacheKey != nil {
		out.PromptCacheKey = strings.TrimSpace(*fileCfg.ResponsePromptCacheKey)
	}
	if fileCfg.ResponsePromptCacheRetention != nil {
		out.PromptCacheRetention = strings.TrimSpace(*fileCfg.ResponsePromptCacheRetention)
	}
	if fileCfg.ResponseSafetyIdentifier != nil {
		out.SafetyIdentifier = strings.TrimSpace(*fileCfg.ResponseSafetyIdentifier)
	}
	if fileCfg.ResponseStore != nil {
		value := *fileCfg.ResponseStore
		out.Store = &value
	}
	if fileCfg.ResponseMetadata != nil {
		normalized, err := normalizeResponseMetadata(fileCfg.ResponseMetadata)
		if err != nil {
			return llm.ResponseOptions{}, err
		}
		out.Metadata = normalized
	}
	return llm.CloneResponseOptions(out), nil
}

func responseMetadataFromEnv(env envSnapshot) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(env))
	for name := range env {
		if strings.HasPrefix(name, responseMetadataEnvPrefix) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	sort.Strings(names)

	raw := make(map[string]string, len(names))
	for _, name := range names {
		key := strings.TrimSpace(strings.TrimPrefix(name, responseMetadataEnvPrefix))
		if key == "" {
			return nil, fmt.Errorf(
				"invalid %s*: metadata key suffix cannot be empty",
				responseMetadataEnvPrefix,
			)
		}
		raw[key] = env.value(name)
	}
	return normalizeResponseMetadata(raw)
}

func parseOptionalBoolValue(raw, fieldName string) (*bool, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := parseBoolValue(raw, fieldName, false)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
