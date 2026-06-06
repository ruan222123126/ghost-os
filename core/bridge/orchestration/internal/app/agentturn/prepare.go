package agentturn

import (
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	bridgeTasks "ghost-os/bridge/tasks"
)

var ErrMessageRequired = errors.New("message or images is required")

func PrepareRequest(params api.AgentParams) (PreparedRequest, error) {
	userInput, message, err := BuildUserInput(params.Message, params.Images)
	if err != nil {
		return PreparedRequest{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	mode, err := NormalizeMode(params.Mode)
	if err != nil {
		return PreparedRequest{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	requestRuntime, err := NormalizeRequestRuntimeOptions(params.ProjectRoot)
	if err != nil {
		return PreparedRequest{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	return PreparedRequest{
		UserInput:      userInput,
		Message:        message,
		Mode:           mode,
		SessionID:      strings.TrimSpace(params.SessionID),
		RequestRuntime: requestRuntime,
	}, nil
}

func BuildUserInput(message string, images []api.SessionImageContent) (llm.Message, string, error) {
	text := strings.TrimSpace(message)
	content, err := normalizeImages(images)
	if err != nil {
		return llm.Message{}, "", err
	}
	if text == "" && len(content) == 0 {
		return llm.Message{}, "", ErrMessageRequired
	}
	return llm.Message{Role: llm.RoleUser, Text: text, Content: content}, text, nil
}

func NormalizeMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return ModeDefault, nil
	}
	if mode == ModePlan {
		return mode, nil
	}
	return "", fmt.Errorf("unsupported agent mode: %q", mode)
}

func NormalizeRequestRuntimeOptions(rawProjectRoot string) (*RequestRuntimeOptions, error) {
	return runtimeopts.NormalizeRequestOptions(rawProjectRoot)
}

func CloneRequestRuntimeOptions(input *RequestRuntimeOptions) *RequestRuntimeOptions {
	return runtimeopts.CloneRequestOptions(input)
}

func PrepareWithRuntimeOverrides(
	guards SessionGuards,
	params api.AgentParams,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) (PreparedRequest, error) {
	prepared, err := PrepareRequest(params)
	if err != nil {
		return PreparedRequest{}, err
	}
	if err := validateSession(guards, prepared.SessionID); err != nil {
		return PreparedRequest{}, err
	}
	prepared.RuntimeOverrides = bridgeTasks.CloneTaskRuntimeOverrides(runtimeOverrides)
	return prepared, nil
}

func validateSession(guards SessionGuards, sessionID string) error {
	if guards == nil {
		return nil
	}
	if err := guards.EnsureSessionNotInflight(sessionID); err != nil {
		return err
	}
	return guards.EnsureSessionActive(sessionID)
}
