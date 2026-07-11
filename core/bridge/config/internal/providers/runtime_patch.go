package providers

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func ApplyRuntimePatch(state State, current RuntimeSnapshot, req RuntimePatchRequest) (Patch, error) {
	state = normalizeState(state)
	if err := applyActiveProviderUpdate(&state, req.Provider); err != nil {
		return Patch{}, err
	}

	index := ActiveProviderIndex(state.Records, state.ActiveProvider, current)
	if index < 0 {
		return state.patch(nil), nil
	}
	updated, changed := applyActiveProviderConfigUpdate(state.Records[index], req)
	if !changed {
		return state.patch(nil), nil
	}
	state.Records[index] = updated
	return state.patch(nil), nil
}

func PrepareRuntimePatchBase(state State, current RuntimeSnapshot, requestedName *string) Patch {
	state = normalizeState(state)
	if len(state.Records) > 0 {
		return state.patch(nil)
	}

	providerName := ActiveLabel(current)
	if requestedName != nil && strings.TrimSpace(*requestedName) != "" {
		providerName = strings.TrimSpace(*requestedName)
	}
	providerType := InferType(providerName, current.BaseURL, current.Model)
	if providerType == "" {
		providerType = defaultProvider
	}

	baseURL := strings.TrimSpace(current.BaseURL)
	if requestedName != nil &&
		strings.TrimSpace(*requestedName) != "" &&
		!strings.EqualFold(providerName, ActiveLabel(current)) {
		baseURL = ""
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL(providerType)
	}

	state.Records = []Record{{
		Name:    providerName,
		Type:    providerType,
		BaseURL: baseURL,
		APIKey:  optionalStringPointer(current.APIKey),
	}}
	state.ActiveProvider = providerName
	return state.patch(nil)
}

func ResolveActive(records []Record, activeName string, fallback RuntimeSnapshot) Record {
	name := strings.TrimSpace(activeName)
	if name == "" {
		name = ActiveLabel(fallback)
	}
	index := IndexByName(records, name)
	if index < 0 {
		return records[0]
	}
	return records[index]
}

func ActiveProviderIndex(records []Record, activeName string, current RuntimeSnapshot) int {
	if index := IndexByName(records, activeName); index >= 0 {
		return index
	}
	if index := IndexByName(records, ActiveLabel(current)); index >= 0 {
		return index
	}
	if len(records) == 0 {
		return -1
	}
	return 0
}

func ActiveLabel(runtime RuntimeSnapshot) string {
	if value := strings.TrimSpace(runtime.ProviderName); value != "" {
		return value
	}
	return string(runtime.Provider)
}

func applyActiveProviderUpdate(state *State, providerName *string) error {
	if providerName == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*providerName)
	if trimmed == "" {
		return ErrNameRequired
	}
	index := IndexByName(state.Records, trimmed)
	if index < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, trimmed)
	}
	state.ActiveProvider = state.Records[index].Name
	return nil
}

func applyActiveProviderConfigUpdate(provider Record, req RuntimePatchRequest) (Record, bool) {
	updated := provider
	changed := false
	if req.APIKey != nil {
		updated.APIKey = cloneOptionalStringPointer(req.APIKey)
		changed = true
	}
	if req.BaseURL != nil {
		updated.BaseURL = resolveUpdatedBaseURL(provider.Type, *req.BaseURL)
		changed = true
	}
	return updated, changed
}

func resolveUpdatedBaseURL(providerType llm.Provider, baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return DefaultBaseURL(providerType)
	}
	return trimmed
}
