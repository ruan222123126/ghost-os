package bus

import (
	"errors"
	"fmt"
	"net/http"
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

type ServiceError struct {
	kind  ServiceErrorKind
	cause error
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return fmt.Sprintf("service error kind=%s", e.kind)
	}
	return e.cause.Error()
}

func (e *ServiceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func WrapError(kind ServiceErrorKind, err error) error {
	if err == nil {
		return nil
	}
	return &ServiceError{kind: kind, cause: err}
}

func ErrorKindOf(err error) ServiceErrorKind {
	if err == nil {
		return ""
	}
	var typed *ServiceError
	if errors.As(err, &typed) && typed != nil {
		return typed.kind
	}
	return ServiceErrorInternal
}

func ResultSuccess(payload any) ServiceResult {
	return ServiceResult{Payload: payload, Outcome: ServiceOutcomeSuccess}
}

func ResultCreated(payload any) ServiceResult {
	return ServiceResult{Payload: payload, Outcome: ServiceOutcomeCreated}
}

func ResultAccepted(payload any) ServiceResult {
	return ServiceResult{Payload: payload, Outcome: ServiceOutcomeAccepted}
}

func OutcomeFromStatus(statusCode int) ServiceOutcome {
	switch statusCode {
	case http.StatusCreated:
		return ServiceOutcomeCreated
	case http.StatusAccepted:
		return ServiceOutcomeAccepted
	default:
		return ServiceOutcomeSuccess
	}
}

func ErrorKindFromStatus(statusCode int) ServiceErrorKind {
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

func StatusFromErrorKind(kind ServiceErrorKind) int {
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
