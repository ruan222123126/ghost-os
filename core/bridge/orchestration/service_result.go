package orchestration

import (
	"net/http"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeskills "ghost-os/bridge/skills"
)

type ServiceOutcome = bus.ServiceOutcome

const (
	ServiceOutcomeSuccess  = bus.ServiceOutcomeSuccess
	ServiceOutcomeCreated  = bus.ServiceOutcomeCreated
	ServiceOutcomeAccepted = bus.ServiceOutcomeAccepted
)

type ServiceErrorKind = bus.ServiceErrorKind

const (
	ServiceErrorInvalidInput = bus.ServiceErrorInvalidInput
	ServiceErrorNotFound     = bus.ServiceErrorNotFound
	ServiceErrorConflict     = bus.ServiceErrorConflict
	ServiceErrorUnavailable  = bus.ServiceErrorUnavailable
	ServiceErrorInternal     = bus.ServiceErrorInternal
)

type ServiceResult = bus.ServiceResult

func wrapServiceError(kind ServiceErrorKind, err error) error {
	return bus.WrapError(kind, err)
}

func ServiceErrorKindOf(err error) ServiceErrorKind {
	return bus.ErrorKindOf(err)
}

func serviceResultSuccess(payload any) ServiceResult {
	return bus.ResultSuccess(payload)
}

func serviceResultCreated(payload any) ServiceResult {
	return bus.ResultCreated(payload)
}

func serviceResultAccepted(payload any) ServiceResult {
	return bus.ResultAccepted(payload)
}

func serviceResultFromStatus(payload any, statusCode int, err error) (ServiceResult, error) {
	if err != nil {
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromStatus(statusCode), err)
	}
	return serviceResultFromStatusSuccess(payload, statusCode), nil
}

func serviceResultFromStatusSuccess(payload any, statusCode int) ServiceResult {
	switch serviceOutcomeFromStatus(statusCode) {
	case ServiceOutcomeCreated:
		return serviceResultCreated(payload)
	case ServiceOutcomeAccepted:
		return serviceResultAccepted(payload)
	default:
		return serviceResultSuccess(payload)
	}
}

func serviceOutcomeFromStatus(statusCode int) ServiceOutcome {
	switch statusCode {
	case http.StatusCreated:
		return ServiceOutcomeCreated
	case http.StatusAccepted:
		return ServiceOutcomeAccepted
	default:
		return ServiceOutcomeSuccess
	}
}

func serviceErrorKindFromStatus(statusCode int) ServiceErrorKind {
	switch statusCode {
	case http.StatusBadRequest:
		return ServiceErrorInvalidInput
	case http.StatusNotFound:
		return ServiceErrorNotFound
	case http.StatusConflict:
		return ServiceErrorConflict
	case http.StatusServiceUnavailable:
		return ServiceErrorUnavailable
	default:
		return ServiceErrorInternal
	}
}

func serviceResultFromLegacySuccess(payload any, statusCode int) ServiceResult {
	return serviceResultFromStatusSuccess(payload, statusCode)
}

func serviceErrorKindFromLegacyStatus(statusCode int) ServiceErrorKind {
	return serviceErrorKindFromStatus(statusCode)
}

func legacyStatusFromServiceErrorKind(kind ServiceErrorKind) int {
	switch kind {
	case ServiceErrorInvalidInput:
		return http.StatusBadRequest
	case ServiceErrorNotFound:
		return http.StatusNotFound
	case ServiceErrorConflict:
		return http.StatusConflict
	case ServiceErrorUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func legacyStatusFromServiceError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return legacyStatusFromServiceErrorKind(ServiceErrorKindOf(err))
}

func truncateRunes(value string, limit int) string {
	return sharedtext.TruncateRunes(value, limit)
}

func adaptLegacyResult(payload any, statusCode int, err error) (ServiceResult, error) {
	return serviceResultFromStatus(payload, statusCode, err)
}

func (s *bridgeService) executeTaskCreateActionResult(params taskCreateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskCreateAction(params, traceID))
}

func (s *bridgeService) executeTaskListActionResult(scope string, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskListAction(scope, traceID))
}

func (s *bridgeService) executeTaskGetActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskGetAction(params, traceID))
}

func (s *bridgeService) executeTaskUpdateActionResult(params taskUpdateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskUpdateAction(params, traceID))
}

func (s *bridgeService) executeTaskRunNowActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskRunNowAction(params, traceID))
}

func (s *bridgeService) executeTaskLogsActionResult(params taskLogsParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskLogsAction(params, traceID))
}

func (s *bridgeService) executeTaskDeleteActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskDeleteAction(params, traceID))
}

func (s *bridgeService) executeSkillListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillListAction(traceID))
}

func (s *bridgeService) executeSkillUpdateActionResult(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillUpdateAction(params, req, traceID))
}

func (s *bridgeService) executeSkillDeleteActionResult(params bridgeskills.SkillIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillDeleteAction(params, traceID))
}

func (s *bridgeService) executeToolListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolListAction(traceID))
}

func (s *bridgeService) executeToolUpdateActionResult(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolUpdateAction(params, req, traceID))
}
