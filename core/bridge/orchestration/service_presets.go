package orchestration

import (
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	appprompts "ghost-os/bridge/orchestration/internal/app/prompts"
)

const (
	presetActionList   = appprompts.PresetActionList
	presetActionCreate = appprompts.PresetActionCreate
	presetActionUpdate = appprompts.PresetActionUpdate
	presetActionDelete = appprompts.PresetActionDelete
	presetActionApply  = appprompts.PresetActionApply
)

func (s *bridgeService) executePresetListAction(traceID string) (ServiceResult, error) {
	return s.promptService().ListPresets(traceID)
}

func (s *bridgeService) executePresetCreateAction(
	req bridgeconfig.PresetCreateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().CreatePreset(req, traceID)
}

func (s *bridgeService) executePresetUpdateAction(
	presetID string,
	req bridgeconfig.PresetUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().UpdatePreset(presetID, req, traceID)
}

func (s *bridgeService) executePresetDeleteAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().DeletePreset(presetID, traceID)
}

func (s *bridgeService) executePresetApplyAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().ApplyPreset(presetID, traceID)
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
