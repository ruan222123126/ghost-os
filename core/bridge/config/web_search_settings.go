package config

import "strings"

type webSearchSettings struct {
	TavilyAPIKey string
	ExaAPIKey    string
	TavilyURL    string
	ExaURL       string
}

func webSearchSettingsFromEnv(env envSnapshot) webSearchSettings {
	return webSearchSettings{
		TavilyAPIKey: env.firstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY"),
		ExaAPIKey:    env.firstNonEmpty("GHOST_WEB_SEARCH_EXA_API_KEY", "EXA_API_KEY"),
		TavilyURL:    env.value("GHOST_WEB_SEARCH_TAVILY_URL"),
		ExaURL:       env.value("GHOST_WEB_SEARCH_EXA_URL"),
	}
}

func fileWebSearchSettings(fileCfg bridgeFileConfig, fallback webSearchSettings) webSearchSettings {
	settings := webSearchSettings{
		TavilyAPIKey: fallback.TavilyAPIKey,
		ExaAPIKey:    fallback.ExaAPIKey,
		TavilyURL:    fallback.TavilyURL,
		ExaURL:       fallback.ExaURL,
	}
	if fileCfg.WebSearchTavilyAPIKey != nil {
		settings.TavilyAPIKey = strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	if fileCfg.WebSearchExaAPIKey != nil {
		settings.ExaAPIKey = strings.TrimSpace(*fileCfg.WebSearchExaAPIKey)
	}
	if fileCfg.WebSearchTavilyURL != nil {
		settings.TavilyURL = strings.TrimSpace(*fileCfg.WebSearchTavilyURL)
	}
	if fileCfg.WebSearchExaURL != nil {
		settings.ExaURL = strings.TrimSpace(*fileCfg.WebSearchExaURL)
	}
	return settings
}
