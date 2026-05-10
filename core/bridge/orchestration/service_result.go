package orchestration

import "ghost-os/bridge/orchestration/internal/contracts/bus"

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
