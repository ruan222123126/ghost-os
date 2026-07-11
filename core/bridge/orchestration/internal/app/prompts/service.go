package prompts

import (
	bridgeconfig "ghost-os/bridge/config"
)

type Store interface {
	Config() (bridgeconfig.Config, error)
	Presets() ([]bridgeconfig.Preset, error)
	CreatePreset(bridgeconfig.PresetCreateRequest) (bridgeconfig.Preset, error)
	UpdatePreset(string, bridgeconfig.PresetUpdateRequest) (bridgeconfig.Preset, error)
	DeletePreset(string) (bridgeconfig.Preset, error)
	ApplyPreset(string) (bridgeconfig.Preset, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type Service struct {
	Store         Store
	PreviewLoader PreviewLoader
	Logger        Logger
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}
