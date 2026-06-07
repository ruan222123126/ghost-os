package storage

import (
	"fmt"
	"strings"
	"time"
)

func ValueOrEnv(raw *string, envName, fallback string) string {
	return ValueOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func ValueOrEnvWithEnv(raw *string, env EnvSnapshot, envName, fallback string) string {
	if raw != nil {
		return strings.TrimSpace(*raw)
	}
	return env.DefaultValue(envName, fallback)
}

func BoolOrEnvWithEnv(raw *bool, env EnvSnapshot, envName string, fallback bool) (bool, error) {
	if raw != nil {
		return *raw, nil
	}
	return ParseBoolValue(env.Value(envName), envName, fallback)
}

func IntOrEnvWithEnv(raw *int, fieldName string, env EnvSnapshot, envName string, fallback int) (int, error) {
	if raw != nil {
		if *raw <= 0 {
			return 0, fmt.Errorf("invalid %s: must be > 0, got %d", resolutionFieldName(fieldName, envName), *raw)
		}
		return *raw, nil
	}
	return ParsePositiveIntValue(env.Value(envName), envName, fallback)
}

func FloatOrEnvWithEnv(raw *float64, fieldName string, env EnvSnapshot, envName string, fallback float64) (float64, error) {
	if raw != nil {
		if *raw < 0 || *raw > 1 {
			return 0, fmt.Errorf(
				"invalid %s: must be between 0 and 1, got %v",
				resolutionFieldName(fieldName, envName),
				*raw,
			)
		}
		return *raw, nil
	}
	return ParseFloatValue(env.Value(envName), envName, fallback)
}

func DurationOrEnvWithEnv(
	raw *string,
	fieldName string,
	env EnvSnapshot,
	envName string,
	fallback time.Duration,
) (time.Duration, error) {
	if raw != nil {
		trimmed := strings.TrimSpace(*raw)
		if trimmed == "" {
			return 0, fmt.Errorf("invalid %s: expected duration, got empty string", resolutionFieldName(fieldName, envName))
		}
		value, err := time.ParseDuration(trimmed)
		if err != nil {
			return 0, fmt.Errorf(
				"invalid %s: expected duration, got %q: %w",
				resolutionFieldName(fieldName, envName),
				trimmed,
				err,
			)
		}
		if value <= 0 {
			return 0, fmt.Errorf("invalid %s: must be > 0, got %q", resolutionFieldName(fieldName, envName), trimmed)
		}
		return value, nil
	}
	return ParseDurationValue(env.Value(envName), envName, fallback)
}

func HeadersOrEnvWithEnv(raw map[string]string, env EnvSnapshot) (map[string]string, error) {
	if len(raw) > 0 {
		return NormalizeProviderHeaders(raw)
	}
	return ParseProviderHeaders(env.Value("GHOST_PROVIDER_HEADERS"))
}

func CORSOriginsOrEnv(raw []string) []string {
	return CORSOriginsOrEnvWithEnv(raw, CurrentEnv())
}

func CORSOriginsOrEnvWithEnv(raw []string, env EnvSnapshot) []string {
	if raw != nil {
		return NormalizeOrigins(raw)
	}
	return ParseOriginsCSV(env.DefaultValue("GHOST_CORS_ORIGINS", ""))
}

func BoundedPositiveIntOrEnvWithEnv(
	raw *int,
	fieldName string,
	env EnvSnapshot,
	envName string,
	fallback int,
	max int,
) (int, error) {
	if raw != nil {
		value := *raw
		if value <= 0 || value > max {
			return 0, invalidBoundedPositiveInt(fieldName, value, max)
		}
		return value, nil
	}

	value, err := ParsePositiveIntValue(env.Value(envName), envName, fallback)
	if err != nil {
		return 0, err
	}
	if value > max {
		return 0, invalidBoundedPositiveInt(envName, value, max)
	}
	return value, nil
}

func resolutionFieldName(fieldName, envName string) string {
	if trimmed := strings.TrimSpace(fieldName); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(envName)
}

func invalidBoundedPositiveInt(name string, value int, max int) error {
	return fmt.Errorf(
		"invalid %s: must be between 1 and %d, got %d",
		strings.TrimSpace(name),
		max,
		value,
	)
}
