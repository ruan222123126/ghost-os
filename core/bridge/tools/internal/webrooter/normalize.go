package webrooter

import (
	"fmt"
	"strings"
)

const responseFieldData = "data"

func NormalizeResponse(action string, payload map[string]any) (Result, error) {
	success, err := requireBool(payload, "success")
	if err != nil {
		return Result{}, actionError(action, err)
	}

	content, err := requireString(payload, "content")
	if err != nil {
		return Result{}, actionError(action, err)
	}

	if err := requireDataField(payload, success); err != nil {
		return Result{}, actionError(action, err)
	}
	if _, err := requireArray(payload, "urls"); err != nil {
		return Result{}, actionError(action, err)
	}
	if _, err := requireObject(payload, "metadata"); err != nil {
		return Result{}, actionError(action, err)
	}

	upstreamError, err := requireNullableString(payload, "error")
	if err != nil {
		return Result{}, actionError(action, err)
	}
	if !success {
		return Result{}, fmt.Errorf(
			"web_rooter %s upstream returned success=false: %s",
			action,
			formatUpstreamFailure(content, upstreamError),
		)
	}

	citations, referencesText, err := normalizeOptionalFields(payload)
	if err != nil {
		return Result{}, actionError(action, err)
	}

	return Result{
		Payload:        payload,
		Citations:      citations,
		ReferencesText: referencesText,
	}, nil
}

func requireDataField(payload map[string]any, success bool) error {
	if !success {
		_, err := requireField(payload, responseFieldData)
		return err
	}

	_, err := requireObject(payload, responseFieldData)
	return err
}

func normalizeOptionalFields(payload map[string]any) ([]any, string, error) {
	citations, err := optionalArray(payload, "citations")
	if err != nil {
		return nil, "", err
	}
	referencesText, err := optionalString(payload, "references_text")
	if err != nil {
		return nil, "", err
	}
	if err := validateOptionalObject(payload, "comparison"); err != nil {
		return nil, "", err
	}
	if err := validateOptionalObject(payload, "mindsearch_compat"); err != nil {
		return nil, "", err
	}
	return citations, referencesText, nil
}

func actionError(action string, err error) error {
	return fmt.Errorf("web_rooter %s %w", action, err)
}

func formatUpstreamFailure(content string, upstreamError string) string {
	trimmedContent := strings.TrimSpace(content)
	trimmedError := strings.TrimSpace(upstreamError)

	switch {
	case trimmedContent != "" && trimmedError != "":
		return fmt.Sprintf("content=%q error=%q", trimmedContent, trimmedError)
	case trimmedError != "":
		return fmt.Sprintf("error=%q", trimmedError)
	case trimmedContent != "":
		return fmt.Sprintf("content=%q", trimmedContent)
	default:
		return "empty upstream error payload"
	}
}

func requireBool(payload map[string]any, key string) (bool, error) {
	value, err := requireField(payload, key)
	if err != nil {
		return false, err
	}

	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("response field %q must be a boolean", key)
	}
	return typed, nil
}

func requireString(payload map[string]any, key string) (string, error) {
	value, err := requireField(payload, key)
	if err != nil {
		return "", err
	}

	typed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("response field %q must be a string", key)
	}
	return typed, nil
}

func requireNullableString(payload map[string]any, key string) (string, error) {
	value, err := requireField(payload, key)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}

	typed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("response field %q must be a string or null", key)
	}
	return typed, nil
}

func requireArray(payload map[string]any, key string) ([]any, error) {
	value, err := requireField(payload, key)
	if err != nil {
		return nil, err
	}

	typed, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("response field %q must be an array", key)
	}
	return typed, nil
}

func requireObject(payload map[string]any, key string) (map[string]any, error) {
	value, err := requireField(payload, key)
	if err != nil {
		return nil, err
	}

	return requireObjectValue(key, value)
}

func requireObjectValue(key string, value any) (map[string]any, error) {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response field %q must be an object", key)
	}
	return typed, nil
}

func requireField(payload map[string]any, key string) (any, error) {
	value, found := payload[key]
	if !found {
		return nil, fmt.Errorf("response field %q is required", key)
	}
	return value, nil
}

func optionalArray(payload map[string]any, key string) ([]any, error) {
	value, found := lookupOptionalField(payload, key)
	if !found {
		return []any{}, nil
	}

	typed, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("response field %q must be an array", key)
	}
	return typed, nil
}

func optionalString(payload map[string]any, key string) (string, error) {
	value, found := lookupOptionalField(payload, key)
	if !found {
		return "", nil
	}

	typed, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("response field %q must be a string", key)
	}
	return typed, nil
}

func validateOptionalObject(payload map[string]any, key string) error {
	value, found := lookupOptionalField(payload, key)
	if !found {
		return nil
	}
	if _, ok := value.(map[string]any); !ok {
		return fmt.Errorf("response field %q must be an object", key)
	}
	return nil
}

func lookupOptionalField(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	if value, ok := payload[key]; ok {
		return value, true
	}

	data, ok := payload[responseFieldData].(map[string]any)
	if !ok {
		return nil, false
	}

	value, found := data[key]
	return value, found
}
