package runtime

import (
	"errors"
	"strings"

	"ghost-os/bridge/config/internal/storage"
)

var ErrModelSelectionDisabled = errors.New("model selection is disabled")

type UpdateRequest struct {
	Model *string
}

type PersistFallbackRequest struct {
	Model              *string
	ChatPath           *string
	ProjectRoot        *string
	WebSearchTavilyURL *string
	WebSearchExaURL    *string
}

func ValidateUpdate(req UpdateRequest, current Snapshot) error {
	if req.Model != nil && !current.ModelSelectionEnabled {
		return ErrModelSelectionDisabled
	}
	return nil
}

func PersistFallback(current Snapshot, req PersistFallbackRequest, env storage.EnvSnapshot) (Snapshot, error) {
	fallback := Clone(current)
	if !hasResetStringField(req) {
		return fallback, nil
	}

	envFallback, err := FallbackFromEnv(env)
	if err != nil {
		return Snapshot{}, err
	}
	if resetsStringValue(req.Model) {
		fallback.Model = envFallback.Model
	}
	if resetsStringValue(req.ChatPath) {
		fallback.ChatPath = envFallback.ChatPath
	}
	if resetsStringValue(req.ProjectRoot) {
		fallback.ProjectRoot = envFallback.ProjectRoot
	}
	if resetsStringValue(req.WebSearchTavilyURL) {
		fallback.WebSearchTavilyURL = envFallback.WebSearchTavilyURL
	}
	if resetsStringValue(req.WebSearchExaURL) {
		fallback.WebSearchExaURL = envFallback.WebSearchExaURL
	}
	return fallback, nil
}

func hasResetStringField(req PersistFallbackRequest) bool {
	return resetsStringValue(req.Model) ||
		resetsStringValue(req.ChatPath) ||
		resetsStringValue(req.ProjectRoot) ||
		resetsStringValue(req.WebSearchTavilyURL) ||
		resetsStringValue(req.WebSearchExaURL)
}

func resetsStringValue(raw *string) bool {
	return raw != nil && strings.TrimSpace(*raw) == ""
}
