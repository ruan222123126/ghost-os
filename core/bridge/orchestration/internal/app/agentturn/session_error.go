package agentturn

import (
	"errors"
	"strings"
)

type SessionError struct {
	SessionID string
	Err       error
}

type SessionSetupError struct {
	SessionID  string
	StatusCode int
	Err        error
}

func (e *SessionSetupError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *SessionSetupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *SessionError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *SessionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func WrapErrorWithSessionID(err error, sessionID string) error {
	if err == nil {
		return nil
	}
	if SessionIDFromError(err) != "" {
		return err
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return err
	}
	return &SessionError{SessionID: trimmedSessionID, Err: err}
}

func SessionIDFromError(err error) string {
	var typed *SessionError
	if !errors.As(err, &typed) || typed == nil {
		return ""
	}
	return strings.TrimSpace(typed.SessionID)
}
