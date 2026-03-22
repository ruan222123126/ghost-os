package config

import (
	"fmt"
	"strings"
)

type webRooterSettings struct {
	Enabled   bool
	BaseURL   string
	APIToken  string
	TimeoutMS int
}

func webRooterSettingsFromEnv(env envSnapshot) (webRooterSettings, error) {
	enabled, err := parseBoolValue(
		env.value("GHOST_WEB_ROOTER_ENABLED"),
		"GHOST_WEB_ROOTER_ENABLED",
		false,
	)
	if err != nil {
		return webRooterSettings{}, err
	}
	timeoutMS, err := parsePositiveIntValue(
		env.value("GHOST_WEB_ROOTER_TIMEOUT_MS"),
		"GHOST_WEB_ROOTER_TIMEOUT_MS",
		defaultWebRooterTimeoutMS,
	)
	if err != nil {
		return webRooterSettings{}, err
	}
	return webRooterSettings{
		Enabled:   enabled,
		BaseURL:   env.defaultValue("GHOST_WEB_ROOTER_BASE_URL", defaultWebRooterBaseURL),
		APIToken:  env.defaultValue("GHOST_WEB_ROOTER_API_TOKEN", ""),
		TimeoutMS: timeoutMS,
	}, nil
}

func fileWebRooterSettings(fileCfg bridgeFileConfig, fallback webRooterSettings) (webRooterSettings, error) {
	settings := webRooterSettings{
		Enabled:   fallback.Enabled,
		BaseURL:   fallback.BaseURL,
		APIToken:  fallback.APIToken,
		TimeoutMS: fallback.TimeoutMS,
	}
	if fileCfg.WebRooterEnabled != nil {
		settings.Enabled = *fileCfg.WebRooterEnabled
	}
	if fileCfg.WebRooterBaseURL != nil {
		settings.BaseURL = strings.TrimSpace(*fileCfg.WebRooterBaseURL)
		if settings.BaseURL == "" {
			return webRooterSettings{}, fmt.Errorf("web_rooter_base_url must not be empty")
		}
	}
	if fileCfg.WebRooterAPIToken != nil {
		settings.APIToken = strings.TrimSpace(*fileCfg.WebRooterAPIToken)
	}
	if fileCfg.WebRooterTimeoutMS != nil {
		if *fileCfg.WebRooterTimeoutMS <= 0 {
			return webRooterSettings{}, fmt.Errorf(
				"invalid web_rooter_timeout_ms: must be > 0, got %d",
				*fileCfg.WebRooterTimeoutMS,
			)
		}
		settings.TimeoutMS = *fileCfg.WebRooterTimeoutMS
	}
	return settings, nil
}
