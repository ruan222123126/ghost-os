package orchestration

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"ghost-os/bridge/llm"
)

const (
	actionConfigProvidersGet      = "CONFIG_PROVIDERS_GET"
	actionConfigProviderCreate    = "CONFIG_PROVIDER_CREATE"
	actionConfigProviderUpdate    = "CONFIG_PROVIDER_UPDATE"
	actionConfigProviderDelete    = "CONFIG_PROVIDER_DELETE"
	actionConfigProviderSetActive = "CONFIG_PROVIDER_SET_ACTIVE"
)

func (s *bridgeService) executeProvidersGetAction(traceID string) (any, int, error) {
	payload := providerListResponse{
		Providers:      buildProviderConfigResponses(s.configStore.ListProviders()),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}
	logAction(traceID, actionConfigProvidersGet, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderCreate, "running", nil)
	modelContextWindowTokens, err := decodeProviderModelTokenOverrides(req.ModelContextWindowTokens)
	if err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	modelResponseReserveTokens, err := decodeProviderModelTokenOverrides(req.ModelResponseReserveTokens)
	if err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	if err := s.configStore.AddProvider(providerConfig{
		Name:                       req.Name,
		Type:                       llm.Provider(req.Type),
		BaseURL:                    stringValue(req.BaseURL),
		APIKey:                     cloneOptionalStringPointer(req.APIKey),
		Models:                     req.Models,
		ContextWindowTokens:        req.ContextWindowTokens,
		ResponseReserveTokens:      req.ResponseReserveTokens,
		ModelContextWindowTokens:   modelContextWindowTokens,
		ModelResponseReserveTokens: modelResponseReserveTokens,
	}); err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	logAction(traceID, actionConfigProviderCreate, "success", nil)
	return providerListResponse{
		Providers:      buildProviderConfigResponses(s.configStore.ListProviders()),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}, http.StatusOK, nil
}

func (s *bridgeService) executeProviderUpdateAction(name string, req providerUpdateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderUpdate, "running", nil)
	modelContextWindowTokens, err := decodeProviderModelTokenOverrides(req.ModelContextWindowTokens)
	if err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	modelResponseReserveTokens, err := decodeProviderModelTokenOverrides(req.ModelResponseReserveTokens)
	if err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	if err := s.configStore.UpdateProvider(name, providerConfig{
		Name:                       req.Name,
		Type:                       llm.Provider(req.Type),
		BaseURL:                    stringValue(req.BaseURL),
		APIKey:                     cloneOptionalStringPointer(req.APIKey),
		Models:                     req.Models,
		ContextWindowTokens:        req.ContextWindowTokens,
		ResponseReserveTokens:      req.ResponseReserveTokens,
		ModelContextWindowTokens:   modelContextWindowTokens,
		ModelResponseReserveTokens: modelResponseReserveTokens,
	}); err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	logAction(traceID, actionConfigProviderUpdate, "success", nil)
	return providerListResponse{
		Providers:      buildProviderConfigResponses(s.configStore.ListProviders()),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}, http.StatusOK, nil
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderDelete, "running", nil)
	if err := s.configStore.DeleteProvider(name); err != nil {
		logAction(traceID, actionConfigProviderDelete, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	logAction(traceID, actionConfigProviderDelete, "success", nil)
	return providerListResponse{
		Providers:      buildProviderConfigResponses(s.configStore.ListProviders()),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}, http.StatusOK, nil
}

func (s *bridgeService) executeSetActiveProviderAction(req setActiveProviderRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderSetActive, "running", nil)
	if err := s.configStore.SetActiveProvider(req.Name); err != nil {
		logAction(traceID, actionConfigProviderSetActive, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	logAction(traceID, actionConfigProviderSetActive, "success", nil)
	return providerListResponse{
		Providers:      buildProviderConfigResponses(s.configStore.ListProviders()),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}, http.StatusOK, nil
}

func buildProviderConfigResponses(providers []providerConfig) []providerConfigResponse {
	if len(providers) == 0 {
		return []providerConfigResponse{}
	}

	out := make([]providerConfigResponse, 0, len(providers))
	for _, provider := range providers {
		out = append(out, providerConfigResponse{
			Name:                       provider.Name,
			Type:                       string(provider.Type),
			BaseURL:                    provider.BaseURL,
			Models:                     append([]string(nil), provider.Models...),
			APIKeySet:                  strings.TrimSpace(stringValue(provider.APIKey)) != "",
			ContextWindowTokens:        provider.ContextWindowTokens,
			ResponseReserveTokens:      provider.ResponseReserveTokens,
			ModelContextWindowTokens:   encodeProviderModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: encodeProviderModelTokenOverrides(provider.ModelResponseReserveTokens),
		})
	}
	return out
}

func decodeProviderModelTokenOverrides(raw map[string]any) (map[string]int, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make(map[string]int, len(raw))
	for key, value := range raw {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}

		parsed, err := decodeProviderModelTokenOverrideValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid model token override %q: %w", normalizedKey, err)
		}
		out[normalizedKey] = parsed
	}
	return out, nil
}

func decodeProviderModelTokenOverrideValue(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int8:
		return int(typed), nil
	case int16:
		return int(typed), nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float32:
		if math.Trunc(float64(typed)) != float64(typed) {
			return 0, fmt.Errorf("expected integer value, got %v", typed)
		}
		return int(typed), nil
	case float64:
		if math.Trunc(typed) != typed {
			return 0, fmt.Errorf("expected integer value, got %v", typed)
		}
		return int(typed), nil
	default:
		return 0, fmt.Errorf("expected integer value, got %T", value)
	}
}

func encodeProviderModelTokenOverrides(raw map[string]int) map[string]any {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]any, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func configProviderStatusCode(err error) int {
	switch {
	case errors.Is(err, errProviderNotFound):
		return http.StatusNotFound
	case errors.Is(err, errProviderExists):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}
