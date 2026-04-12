package orchestration

import "net/http"

func serviceResultFromLegacy(payload any, statusCode int, err error) (ServiceResult, error) {
	if err != nil {
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromLegacyStatus(statusCode), err)
	}
	return serviceResultFromLegacySuccess(payload, statusCode), nil
}

func serviceResultFromLegacySuccess(payload any, statusCode int) ServiceResult {
	switch serviceOutcomeFromLegacyStatus(statusCode) {
	case ServiceOutcomeCreated:
		return serviceResultCreated(payload)
	case ServiceOutcomeAccepted:
		return serviceResultAccepted(payload)
	default:
		return serviceResultSuccess(payload)
	}
}

func serviceOutcomeFromLegacyStatus(statusCode int) ServiceOutcome {
	switch statusCode {
	case http.StatusCreated:
		return ServiceOutcomeCreated
	case http.StatusAccepted:
		return ServiceOutcomeAccepted
	default:
		return ServiceOutcomeSuccess
	}
}

func serviceErrorKindFromLegacyStatus(statusCode int) ServiceErrorKind {
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
