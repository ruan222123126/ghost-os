package orchestration

import (
	"errors"
	"fmt"
)

type ServiceOutcome string

const (
	ServiceOutcomeSuccess  ServiceOutcome = "success"
	ServiceOutcomeCreated  ServiceOutcome = "created"
	ServiceOutcomeAccepted ServiceOutcome = "accepted"
)

type ServiceErrorKind string

const (
	ServiceErrorInvalidInput ServiceErrorKind = "invalid_input"
	ServiceErrorNotFound     ServiceErrorKind = "not_found"
	ServiceErrorConflict     ServiceErrorKind = "conflict"
	ServiceErrorUnavailable  ServiceErrorKind = "unavailable"
	ServiceErrorInternal     ServiceErrorKind = "internal"
)

type ServiceResult struct {
	Payload any
	Outcome ServiceOutcome
}

type serviceError struct {
	kind  ServiceErrorKind
	cause error
}

func (e *serviceError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return fmt.Sprintf("service error kind=%s", e.kind)
	}
	return e.cause.Error()
}

func (e *serviceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func wrapServiceError(kind ServiceErrorKind, err error) error {
	if err == nil {
		return nil
	}
	return &serviceError{
		kind:  kind,
		cause: err,
	}
}

func ServiceErrorKindOf(err error) ServiceErrorKind {
	if err == nil {
		return ""
	}
	var typed *serviceError
	if errors.As(err, &typed) && typed != nil {
		return typed.kind
	}
	return ServiceErrorInternal
}

func serviceResultSuccess(payload any) ServiceResult {
	return ServiceResult{
		Payload: payload,
		Outcome: ServiceOutcomeSuccess,
	}
}

func serviceResultCreated(payload any) ServiceResult {
	return ServiceResult{
		Payload: payload,
		Outcome: ServiceOutcomeCreated,
	}
}

func serviceResultAccepted(payload any) ServiceResult {
	return ServiceResult{
		Payload: payload,
		Outcome: ServiceOutcomeAccepted,
	}
}
