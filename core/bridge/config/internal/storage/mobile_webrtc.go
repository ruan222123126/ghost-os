package storage

import (
	"encoding/json"
	"fmt"
	"strings"
)

type MobileICEServerResolved struct {
	URLs       []string
	Username   string
	Credential string
}

type MobileWebRTCResolvedConfig struct {
	Enabled             bool
	SignalingURL        string
	SignalingToken      string
	PCID                string
	ICEServers          []MobileICEServerResolved
	CredentialStorePath string
}

func ResolveMobileWebRTCConfig(
	fileCfg FileConfig,
	env EnvSnapshot,
	defaultCredentialStorePath string,
) (MobileWebRTCResolvedConfig, error) {
	raw := fileCfg.MobileWebRTC
	enabled, err := BoolOrEnvWithEnv(raw.Enabled, env, "GHOST_MOBILE_WEBRTC_ENABLED", false)
	if err != nil {
		return MobileWebRTCResolvedConfig{}, err
	}

	iceServers, err := resolveMobileICEServers(raw.ICEServers, env)
	if err != nil {
		return MobileWebRTCResolvedConfig{}, err
	}

	storePath, err := ResolveUserPath(ValueOrEnvWithEnv(
		raw.CredentialStorePath,
		env,
		"GHOST_MOBILE_CREDENTIAL_STORE_PATH",
		defaultCredentialStorePath,
	))
	if err != nil {
		return MobileWebRTCResolvedConfig{}, fmt.Errorf("resolve mobile_webrtc.credential_store_path: %w", err)
	}

	cfg := MobileWebRTCResolvedConfig{
		Enabled:             enabled,
		SignalingURL:        ValueOrEnvWithEnv(raw.SignalingURL, env, "GHOST_SIGNALING_URL", ""),
		SignalingToken:      ValueOrEnvWithEnv(raw.SignalingToken, env, "GHOST_SIGNALING_TOKEN", ""),
		PCID:                ValueOrEnvWithEnv(raw.PCID, env, "GHOST_MOBILE_PC_ID", ""),
		ICEServers:          iceServers,
		CredentialStorePath: storePath,
	}
	if err := validateMobileWebRTCConfig(cfg); err != nil {
		return MobileWebRTCResolvedConfig{}, err
	}
	return cfg, nil
}

func resolveMobileICEServers(raw []MobileICEFileConfig, env EnvSnapshot) ([]MobileICEServerResolved, error) {
	if override := env.Value("GHOST_ICE_SERVERS_JSON"); override != "" {
		return parseMobileICEServersJSON(override)
	}
	return mobileICEServersFromFile(raw), nil
}

func parseMobileICEServersJSON(raw string) ([]MobileICEServerResolved, error) {
	var decoded []mobileICEJSONConfig
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("invalid GHOST_ICE_SERVERS_JSON: %w", err)
	}
	return normalizeResolvedMobileICEServers(decoded), nil
}

func mobileICEServersFromFile(raw []MobileICEFileConfig) []MobileICEServerResolved {
	if len(raw) == 0 {
		return nil
	}
	values := make([]mobileICEJSONConfig, 0, len(raw))
	for _, server := range raw {
		values = append(values, mobileICEJSONConfig{
			URLs:       mobileICEURLList(append([]string(nil), server.URLs...)),
			Username:   StringValue(server.Username),
			Credential: StringValue(server.Credential),
		})
	}
	return normalizeResolvedMobileICEServers(values)
}

type mobileICEJSONConfig struct {
	URLs       mobileICEURLList `json:"urls"`
	Username   string           `json:"username"`
	Credential string           `json:"credential"`
}

type mobileICEURLList []string

func (l *mobileICEURLList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*l = mobileICEURLList{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*l = mobileICEURLList(many)
	return nil
}

func normalizeResolvedMobileICEServers(raw []mobileICEJSONConfig) []MobileICEServerResolved {
	servers := make([]MobileICEServerResolved, 0, len(raw))
	for _, server := range raw {
		urls := NormalizeStringList([]string(server.URLs))
		if len(urls) == 0 {
			continue
		}
		servers = append(servers, MobileICEServerResolved{
			URLs:       urls,
			Username:   strings.TrimSpace(server.Username),
			Credential: strings.TrimSpace(server.Credential),
		})
	}
	if len(servers) == 0 {
		return nil
	}
	return servers
}

func validateMobileWebRTCConfig(cfg MobileWebRTCResolvedConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.SignalingURL) == "" {
		return fmt.Errorf("mobile_webrtc.signaling_url is required when mobile_webrtc.enabled=true")
	}
	if strings.TrimSpace(cfg.SignalingToken) == "" {
		return fmt.Errorf("mobile_webrtc.signaling_token is required when mobile_webrtc.enabled=true")
	}
	if strings.TrimSpace(cfg.PCID) == "" {
		return fmt.Errorf("mobile_webrtc.pc_id is required when mobile_webrtc.enabled=true")
	}
	if len(cfg.ICEServers) == 0 {
		return fmt.Errorf("mobile_webrtc.ice_servers is required when mobile_webrtc.enabled=true")
	}
	if strings.TrimSpace(cfg.CredentialStorePath) == "" {
		return fmt.Errorf("mobile_webrtc.credential_store_path is required when mobile_webrtc.enabled=true")
	}
	return nil
}
