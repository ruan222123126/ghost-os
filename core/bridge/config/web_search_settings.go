package config

import "strings"

type webSearchSettings struct {
	TavilyAPIKey string
	ExaAPIKey    string
}

func envWebSearchSettings() webSearchSettings {
	return webSearchSettingsFromEnv(CurrentEnv())
}

func webSearchSettingsFromEnv(env Env) webSearchSettings {
	return webSearchSettings{
		TavilyAPIKey: env.firstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY"),
		ExaAPIKey:    env.firstNonEmpty("GHOST_WEB_SEARCH_EXA_API_KEY", "EXA_API_KEY"),
	}
}

func fileWebSearchSettings(fileCfg bridgeFileConfig, fallback webSearchSettings) webSearchSettings {
	settings := webSearchSettings{
		TavilyAPIKey: fallback.TavilyAPIKey,
		ExaAPIKey:    fallback.ExaAPIKey,
	}
	if fileCfg.WebSearchTavilyAPIKey != nil {
		settings.TavilyAPIKey = strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchExaAPIKey != nil {
		settings.ExaAPIKey = strings.TrimSpace(*fileCfg.WebSearchExaAPIKey)
	}
	return settings
}
