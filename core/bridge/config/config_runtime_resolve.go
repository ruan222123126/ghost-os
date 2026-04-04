package config

import "ghost-os/bridge/llm"

func runtimeConfigFromEnv() (runtimeConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return runtimeConfig{}, err
	}
	return resolveRuntimeConfig(fileCfg, currentEnv())
}

func resolveRuntimeConfig(fileCfg bridgeFileConfig, env envSnapshot) (runtimeConfig, error) {
	fallback, err := runtimeFallbackFromEnv(env)
	if err != nil {
		return runtimeConfig{}, err
	}
	return resolveRuntimeConfigWithFallback(fileCfg, fallback)
}

func runtimeFallbackFromEnv(env envSnapshot) (runtimeConfig, error) {
	settings, err := resolveRuntimeFallbackSettings(env)
	if err != nil {
		return runtimeConfig{}, err
	}
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               env.defaultValue("GHOST_PROVIDER", string(defaultProvider)),
		APIKey:                     env.defaultValue("GHOST_API_KEY", ""),
		BaseURL:                    env.defaultValue("GHOST_BASE_URL", ""),
		Model:                      env.defaultValue("GHOST_MODEL", ""),
		ChatPath:                   env.defaultValue("GHOST_CHAT_PATH", ""),
		ResponseOptions:            settings.ResponseOptions,
		CodexStatelessRetryEnabled: settings.CodexStatelessRetryEnabled,
		NativePersistent:           settings.NativePersistent,
		ProjectRoot:                env.defaultValue("GHOST_PROJECT_ROOT", ""),
		ModelSelectionEnabled:      !settings.AllowlistOnly,
		WebSearchTavilyURL:         settings.WebSearch.TavilyURL,
		WebSearchExaURL:            settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:      settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         settings.WebSearch.ExaAPIKey,
		WebRooterEnabled:           settings.WebRooter.Enabled,
		WebRooterBaseURL:           settings.WebRooter.BaseURL,
		WebRooterAPIToken:          settings.WebRooter.APIToken,
		WebRooterTimeoutMS:         settings.WebRooter.TimeoutMS,
		GraphQL:                    settings.GraphQL,
	}), nil
}

type runtimeFallbackSettings struct {
	WebSearch                  webSearchSettings
	WebRooter                  webRooterSettings
	ResponseOptions            llm.ResponseOptions
	GraphQL                    GraphQLConfig
	CodexStatelessRetryEnabled bool
	AllowlistOnly              bool
	NativePersistent           bool
}

func resolveRuntimeFallbackSettings(env envSnapshot) (runtimeFallbackSettings, error) {
	webRooter, err := webRooterSettingsFromEnv(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	responseOptions, err := responseOptionsFromEnv(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	graphQL, err := graphQLSettingsFromEnv(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	codexRetryEnabled, allowlistOnly, nativePersistent, err := resolveRuntimeFallbackFlags(env)
	if err != nil {
		return runtimeFallbackSettings{}, err
	}
	return runtimeFallbackSettings{
		WebSearch:                  webSearchSettingsFromEnv(env),
		WebRooter:                  webRooter,
		ResponseOptions:            responseOptions,
		GraphQL:                    graphQL,
		CodexStatelessRetryEnabled: codexRetryEnabled,
		AllowlistOnly:              allowlistOnly,
		NativePersistent:           nativePersistent,
	}, nil
}

func resolveRuntimeFallbackFlags(env envSnapshot) (bool, bool, bool, error) {
	codexRetryEnabled, err := parseBoolValue(
		env.value("GHOST_CODEX_STATELESS_RETRY_ENABLED"),
		"GHOST_CODEX_STATELESS_RETRY_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, err
	}
	allowlistOnly, err := parseBoolValue(env.value("GHOST_TOOL_ALLOWLIST_ONLY"), "GHOST_TOOL_ALLOWLIST_ONLY", false)
	if err != nil {
		return false, false, false, err
	}
	nativePersistent, err := resolveNativePersistent(nil, env)
	if err != nil {
		return false, false, false, err
	}
	return codexRetryEnabled, allowlistOnly, nativePersistent, nil
}

// resolveRuntimeConfigWithFallback folds file overrides onto an existing
// runtime snapshot. Store patch flows use this explicit exception path.
func resolveRuntimeConfigWithFallback(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeConfig, error) {
	normalizedFileCfg, err := normalizeBridgeFileConfigForWrite(fileCfg)
	if err != nil {
		return runtimeConfig{}, err
	}
	fallback = normalizeRuntimeConfig(fallback)
	settings, err := resolveRuntimeFileSettings(normalizedFileCfg, fallback)
	if err != nil {
		return runtimeConfig{}, err
	}
	if len(settings.Providers) > 0 {
		return runtimeConfigWithProviders(
			normalizedFileCfg,
			fallback,
			settings.Providers,
			settings.AllowlistOnly,
			settings.WebSearch,
			settings.WebRooter,
			settings.ResponseOptions,
			settings.CodexStatelessRetryEnabled,
			settings.GraphQL,
		), nil
	}
	return runtimeConfigWithoutProviders(
		normalizedFileCfg,
		fallback,
		settings.AllowlistOnly,
		settings.WebSearch,
		settings.WebRooter,
		settings.ResponseOptions,
		settings.CodexStatelessRetryEnabled,
		settings.GraphQL,
	), nil
}

type runtimeFileSettings struct {
	WebSearch                  webSearchSettings
	WebRooter                  webRooterSettings
	ResponseOptions            llm.ResponseOptions
	GraphQL                    GraphQLConfig
	Providers                  []providerConfig
	CodexStatelessRetryEnabled bool
	AllowlistOnly              bool
}

func resolveRuntimeFileSettings(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeFileSettings, error) {
	webRooter, err := fileWebRooterSettings(fileCfg, webRooterSettings{
		Enabled:   fallback.WebRooterEnabled,
		BaseURL:   fallback.WebRooterBaseURL,
		APIToken:  fallback.WebRooterAPIToken,
		TimeoutMS: fallback.WebRooterTimeoutMS,
	})
	if err != nil {
		return runtimeFileSettings{}, err
	}
	responseOptions, err := fileResponseOptions(fileCfg, fallback.ResponseOptions)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	graphQL, err := fileGraphQLSettings(fileCfg, fallback.GraphQL)
	if err != nil {
		return runtimeFileSettings{}, err
	}
	return runtimeFileSettings{
		WebSearch: fileWebSearchSettings(fileCfg, webSearchSettings{
			TavilyURL:    fallback.WebSearchTavilyURL,
			ExaURL:       fallback.WebSearchExaURL,
			TavilyAPIKey: fallback.WebSearchTavilyAPIKey,
			ExaAPIKey:    fallback.WebSearchExaAPIKey,
		}),
		WebRooter:                  webRooter,
		ResponseOptions:            responseOptions,
		GraphQL:                    graphQL,
		Providers:                  normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model)),
		CodexStatelessRetryEnabled: resolveRuntimeCodexRetryEnabled(fileCfg, fallback),
		AllowlistOnly:              resolveRuntimeAllowlistOnly(fileCfg, fallback),
	}, nil
}

func resolveRuntimeCodexRetryEnabled(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.CodexStatelessRetryEnabled != nil {
		return *fileCfg.CodexStatelessRetryEnabled
	}
	return fallback.CodexStatelessRetryEnabled
}

func resolveRuntimeAllowlistOnly(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.ToolAllowlistOnly != nil {
		return *fileCfg.ToolAllowlistOnly
	}
	return !fallback.ModelSelectionEnabled
}

func runtimeConfigWithProviders(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
	providers []providerConfig,
	allowlistOnly bool,
	webSearch webSearchSettings,
	webRooter webRooterSettings,
	responseOptions llm.ResponseOptions,
	codexRetryEnabled bool,
	graphql GraphQLConfig,
) runtimeConfig {
	active := resolveActiveProvider(providers, stringValue(fileCfg.ActiveProvider), fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               active.Name,
		Provider:                   active.Type.Normalized(),
		APIKey:                     resolveRuntimeAPIKey(active, fallback.APIKey),
		BaseURL:                    resolveRuntimeBaseURL(active.BaseURL, fallback.BaseURL),
		Model:                      resolveRuntimeModel(fileCfg, fallback),
		ChatPath:                   resolveRuntimeChatPath(fileCfg, fallback),
		ResponseOptions:            responseOptions,
		CodexStatelessRetryEnabled: codexRetryEnabled,
		NativePersistent:           resolveRuntimeNativePersistent(fileCfg, fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(fileCfg, fallback),
		ModelSelectionEnabled:      !allowlistOnly,
		ContextWindowTokens:        active.ContextWindowTokens,
		ResponseReserveTokens:      active.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(active.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(active.ModelResponseReserveTokens),
		WebSearchTavilyURL:         webSearch.TavilyURL,
		WebSearchExaURL:            webSearch.ExaURL,
		WebSearchTavilyAPIKey:      webSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         webSearch.ExaAPIKey,
		WebRooterEnabled:           webRooter.Enabled,
		WebRooterBaseURL:           webRooter.BaseURL,
		WebRooterAPIToken:          webRooter.APIToken,
		WebRooterTimeoutMS:         webRooter.TimeoutMS,
		GraphQL:                    graphql,
	})
}

func runtimeConfigWithoutProviders(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
	allowlistOnly bool,
	webSearch webSearchSettings,
	webRooter webRooterSettings,
	responseOptions llm.ResponseOptions,
	codexRetryEnabled bool,
	graphql GraphQLConfig,
) runtimeConfig {
	providerName := resolveRuntimeProviderName(fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               providerName,
		Provider:                   inferProviderType(providerName, fallback.BaseURL, resolveRuntimeModel(fileCfg, fallback)),
		APIKey:                     fallback.APIKey,
		BaseURL:                    fallback.BaseURL,
		Model:                      resolveRuntimeModel(fileCfg, fallback),
		ChatPath:                   resolveRuntimeChatPath(fileCfg, fallback),
		ResponseOptions:            responseOptions,
		CodexStatelessRetryEnabled: codexRetryEnabled,
		NativePersistent:           resolveRuntimeNativePersistent(fileCfg, fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(fileCfg, fallback),
		ModelSelectionEnabled:      !allowlistOnly,
		ContextWindowTokens:        fallback.ContextWindowTokens,
		ResponseReserveTokens:      fallback.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(fallback.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(fallback.ModelResponseReserveTokens),
		WebSearchTavilyURL:         webSearch.TavilyURL,
		WebSearchExaURL:            webSearch.ExaURL,
		WebSearchTavilyAPIKey:      webSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         webSearch.ExaAPIKey,
		WebRooterEnabled:           webRooter.Enabled,
		WebRooterBaseURL:           webRooter.BaseURL,
		WebRooterAPIToken:          webRooter.APIToken,
		WebRooterTimeoutMS:         webRooter.TimeoutMS,
		GraphQL:                    graphql,
	})
}
