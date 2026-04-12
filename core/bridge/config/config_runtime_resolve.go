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
		SessionHumanLogFullEnabled: settings.SessionHumanLogFullEnabled,
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
	SessionHumanLogFullEnabled bool
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
	codexRetryEnabled, allowlistOnly, nativePersistent, sessionHumanLogFullEnabled, err := resolveRuntimeFallbackFlags(env)
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
		SessionHumanLogFullEnabled: sessionHumanLogFullEnabled,
	}, nil
}

func resolveRuntimeFallbackFlags(env envSnapshot) (bool, bool, bool, bool, error) {
	codexRetryEnabled, err := parseBoolValue(
		env.value("GHOST_CODEX_STATELESS_RETRY_ENABLED"),
		"GHOST_CODEX_STATELESS_RETRY_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, false, err
	}
	allowlistOnly, err := parseBoolValue(env.value("GHOST_TOOL_ALLOWLIST_ONLY"), "GHOST_TOOL_ALLOWLIST_ONLY", false)
	if err != nil {
		return false, false, false, false, err
	}
	nativePersistent, err := resolveNativePersistent(nil, env)
	if err != nil {
		return false, false, false, false, err
	}
	sessionHumanLogFullEnabled, err := parseBoolValue(
		env.value("GHOST_SESSION_HUMAN_LOG_FULL_ENABLED"),
		"GHOST_SESSION_HUMAN_LOG_FULL_ENABLED",
		false,
	)
	if err != nil {
		return false, false, false, false, err
	}
	return codexRetryEnabled, allowlistOnly, nativePersistent, sessionHumanLogFullEnabled, nil
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
	buildInput := runtimeConfigBuildInput{
		FileCfg:  normalizedFileCfg,
		Fallback: fallback,
		Settings: settings,
	}
	if len(settings.Providers) > 0 {
		return runtimeConfigWithProviders(buildInput, settings.Providers), nil
	}
	return runtimeConfigWithoutProviders(buildInput), nil
}

type runtimeFileSettings struct {
	WebSearch                  webSearchSettings
	WebRooter                  webRooterSettings
	ResponseOptions            llm.ResponseOptions
	GraphQL                    GraphQLConfig
	Providers                  []providerConfig
	CodexStatelessRetryEnabled bool
	AllowlistOnly              bool
	SessionHumanLogFullEnabled bool
}

type runtimeConfigBuildInput struct {
	FileCfg  bridgeFileConfig
	Fallback runtimeConfig
	Settings runtimeFileSettings
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
		SessionHumanLogFullEnabled: resolveRuntimeSessionHumanLogFullEnabled(fileCfg, fallback),
	}, nil
}

func runtimeConfigWithProviders(input runtimeConfigBuildInput, providers []providerConfig) runtimeConfig {
	active := resolveActiveProvider(providers, stringValue(input.FileCfg.ActiveProvider), input.Fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               active.Name,
		Provider:                   active.Type.Normalized(),
		APIKey:                     resolveRuntimeAPIKey(active, input.Fallback.APIKey),
		BaseURL:                    resolveRuntimeBaseURL(active.BaseURL, input.Fallback.BaseURL),
		Model:                      resolveRuntimeModel(input.FileCfg, input.Fallback),
		ChatPath:                   resolveRuntimeChatPath(input.FileCfg, input.Fallback),
		ResponseOptions:            input.Settings.ResponseOptions,
		CodexStatelessRetryEnabled: input.Settings.CodexStatelessRetryEnabled,
		NativePersistent:           resolveRuntimeNativePersistent(input.FileCfg, input.Fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(input.FileCfg, input.Fallback),
		ModelSelectionEnabled:      !input.Settings.AllowlistOnly,
		ContextWindowTokens:        active.ContextWindowTokens,
		ResponseReserveTokens:      active.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(active.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(active.ModelResponseReserveTokens),
		WebSearchTavilyURL:         input.Settings.WebSearch.TavilyURL,
		WebSearchExaURL:            input.Settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:      input.Settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         input.Settings.WebSearch.ExaAPIKey,
		SessionHumanLogFullEnabled: input.Settings.SessionHumanLogFullEnabled,
		WebRooterEnabled:           input.Settings.WebRooter.Enabled,
		WebRooterBaseURL:           input.Settings.WebRooter.BaseURL,
		WebRooterAPIToken:          input.Settings.WebRooter.APIToken,
		WebRooterTimeoutMS:         input.Settings.WebRooter.TimeoutMS,
		GraphQL:                    input.Settings.GraphQL,
	})
}

func runtimeConfigWithoutProviders(input runtimeConfigBuildInput) runtimeConfig {
	providerName := resolveRuntimeProviderName(input.Fallback)
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:               providerName,
		Provider:                   inferProviderType(providerName, input.Fallback.BaseURL, resolveRuntimeModel(input.FileCfg, input.Fallback)),
		APIKey:                     input.Fallback.APIKey,
		BaseURL:                    input.Fallback.BaseURL,
		Model:                      resolveRuntimeModel(input.FileCfg, input.Fallback),
		ChatPath:                   resolveRuntimeChatPath(input.FileCfg, input.Fallback),
		ResponseOptions:            input.Settings.ResponseOptions,
		CodexStatelessRetryEnabled: input.Settings.CodexStatelessRetryEnabled,
		NativePersistent:           resolveRuntimeNativePersistent(input.FileCfg, input.Fallback),
		ProjectRoot:                resolveRuntimeProjectRoot(input.FileCfg, input.Fallback),
		ModelSelectionEnabled:      !input.Settings.AllowlistOnly,
		ContextWindowTokens:        input.Fallback.ContextWindowTokens,
		ResponseReserveTokens:      input.Fallback.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(input.Fallback.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(input.Fallback.ModelResponseReserveTokens),
		WebSearchTavilyURL:         input.Settings.WebSearch.TavilyURL,
		WebSearchExaURL:            input.Settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:      input.Settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:         input.Settings.WebSearch.ExaAPIKey,
		SessionHumanLogFullEnabled: input.Settings.SessionHumanLogFullEnabled,
		WebRooterEnabled:           input.Settings.WebRooter.Enabled,
		WebRooterBaseURL:           input.Settings.WebRooter.BaseURL,
		WebRooterAPIToken:          input.Settings.WebRooter.APIToken,
		WebRooterTimeoutMS:         input.Settings.WebRooter.TimeoutMS,
		GraphQL:                    input.Settings.GraphQL,
	})
}
