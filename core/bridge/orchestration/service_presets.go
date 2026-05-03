package orchestration

import (
	"errors"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

const (
	presetActionList   = "PRESET_LIST"
	presetActionCreate = "PRESET_CREATE"
	presetActionUpdate = "PRESET_UPDATE"
	presetActionDelete = "PRESET_DELETE"
	presetActionApply  = "PRESET_APPLY"
)

func (s *bridgeService) executePresetListAction(traceID string) (ServiceResult, error) {
	presets, err := s.configStore.Presets()
	if err != nil {
		logAction(traceID, presetActionList, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, presetActionList, "success", nil)
	return serviceResultSuccess(presets), nil
}

func (s *bridgeService) executePresetCreateAction(
	req bridgeconfig.PresetCreateRequest,
	traceID string,
) (ServiceResult, error) {
	preset, err := s.configStore.CreatePreset(req)
	if err != nil {
		logAction(traceID, presetActionCreate, "error", err)
		return ServiceResult{}, mapPresetError(err)
	}
	logAction(traceID, presetActionCreate, "success", nil)
	return serviceResultCreated(preset), nil
}

func (s *bridgeService) executePresetUpdateAction(
	presetID string,
	req bridgeconfig.PresetUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	preset, err := s.configStore.UpdatePreset(strings.TrimSpace(presetID), req)
	if err != nil {
		logAction(traceID, presetActionUpdate, "error", err)
		return ServiceResult{}, mapPresetError(err)
	}
	logAction(traceID, presetActionUpdate, "success", nil)
	return serviceResultSuccess(preset), nil
}

func (s *bridgeService) executePresetDeleteAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	preset, err := s.configStore.DeletePreset(strings.TrimSpace(presetID))
	if err != nil {
		logAction(traceID, presetActionDelete, "error", err)
		return ServiceResult{}, mapPresetError(err)
	}
	logAction(traceID, presetActionDelete, "success", nil)
	return serviceResultSuccess(preset), nil
}

func (s *bridgeService) executePresetApplyAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	preset, err := s.configStore.ApplyPreset(strings.TrimSpace(presetID))
	if err != nil {
		logAction(traceID, presetActionApply, "error", err)
		return ServiceResult{}, mapPresetError(err)
	}
	logAction(traceID, presetActionApply, "success", nil)
	return serviceResultSuccess(preset), nil
}

func mapPresetError(err error) error {
	switch {
	case errors.Is(err, bridgeconfig.ErrPresetNotFound):
		return wrapServiceError(ServiceErrorNotFound, err)
	case errors.Is(err, bridgeconfig.ErrPresetInvalid),
		errors.Is(err, bridgeconfig.ErrPresetIDRequired),
		errors.Is(err, bridgeconfig.ErrPresetUpdateEmpty):
		return wrapServiceError(ServiceErrorInvalidInput, err)
	default:
		return wrapServiceError(ServiceErrorInternal, err)
	}
}
