package prompts

import (
	"errors"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

const (
	PresetActionList   = "PRESET_LIST"
	PresetActionCreate = "PRESET_CREATE"
	PresetActionUpdate = "PRESET_UPDATE"
	PresetActionDelete = "PRESET_DELETE"
	PresetActionApply  = "PRESET_APPLY"
)

func (s Service) ListPresets(traceID string) (bus.ServiceResult, error) {
	presets, err := s.Store.Presets()
	if err != nil {
		s.log(traceID, PresetActionList, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.log(traceID, PresetActionList, "success", nil)
	return bus.ResultSuccess(presets), nil
}

func (s Service) CreatePreset(
	req bridgeconfig.PresetCreateRequest,
	traceID string,
) (bus.ServiceResult, error) {
	preset, err := s.Store.CreatePreset(req)
	if err != nil {
		s.log(traceID, PresetActionCreate, "error", err)
		return bus.ServiceResult{}, mapPresetError(err)
	}
	s.log(traceID, PresetActionCreate, "success", nil)
	return bus.ResultCreated(preset), nil
}

func (s Service) UpdatePreset(
	presetID string,
	req bridgeconfig.PresetUpdateRequest,
	traceID string,
) (bus.ServiceResult, error) {
	preset, err := s.Store.UpdatePreset(strings.TrimSpace(presetID), req)
	if err != nil {
		s.log(traceID, PresetActionUpdate, "error", err)
		return bus.ServiceResult{}, mapPresetError(err)
	}
	s.log(traceID, PresetActionUpdate, "success", nil)
	return bus.ResultSuccess(preset), nil
}

func (s Service) DeletePreset(presetID string, traceID string) (bus.ServiceResult, error) {
	preset, err := s.Store.DeletePreset(strings.TrimSpace(presetID))
	if err != nil {
		s.log(traceID, PresetActionDelete, "error", err)
		return bus.ServiceResult{}, mapPresetError(err)
	}
	s.log(traceID, PresetActionDelete, "success", nil)
	return bus.ResultSuccess(preset), nil
}

func (s Service) ApplyPreset(presetID string, traceID string) (bus.ServiceResult, error) {
	preset, err := s.Store.ApplyPreset(strings.TrimSpace(presetID))
	if err != nil {
		s.log(traceID, PresetActionApply, "error", err)
		return bus.ServiceResult{}, mapPresetError(err)
	}
	s.log(traceID, PresetActionApply, "success", nil)
	return bus.ResultSuccess(preset), nil
}

func mapPresetError(err error) error {
	switch {
	case errors.Is(err, bridgeconfig.ErrPresetNotFound):
		return bus.WrapError(bus.ServiceErrorNotFound, err)
	case errors.Is(err, bridgeconfig.ErrPresetInvalid),
		errors.Is(err, bridgeconfig.ErrPresetIDRequired),
		errors.Is(err, bridgeconfig.ErrPresetUpdateEmpty):
		return bus.WrapError(bus.ServiceErrorInvalidInput, err)
	default:
		return bus.WrapError(bus.ServiceErrorInternal, err)
	}
}
