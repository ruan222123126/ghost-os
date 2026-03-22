package orchestration

import (
	"errors"
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
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProvidersGet, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionConfigProvidersGet, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderCreate, "running", nil)
	if err := s.configStore.AddProvider(providerConfig{
		Name:                       req.Name,
		Type:                       llm.Provider(req.Type),
		BaseURL:                    stringValue(req.BaseURL),
		APIKey:                     cloneOptionalStringPointer(req.APIKey),
		Models:                     req.Models,
		ContextWindowTokens:        req.ContextWindowTokens,
		ResponseReserveTokens:      req.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(req.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(req.ModelResponseReserveTokens),
	}); err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionConfigProviderCreate, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeProviderUpdateAction(name string, req providerUpdateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderUpdate, "running", nil)
	if err := s.configStore.UpdateProvider(name, providerConfig{
		Name:                       req.Name,
		Type:                       llm.Provider(req.Type),
		BaseURL:                    stringValue(req.BaseURL),
		APIKey:                     cloneOptionalStringPointer(req.APIKey),
		Models:                     req.Models,
		ContextWindowTokens:        req.ContextWindowTokens,
		ResponseReserveTokens:      req.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(req.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(req.ModelResponseReserveTokens),
	}); err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionConfigProviderUpdate, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderDelete, "running", nil)
	if err := s.configStore.DeleteProvider(name); err != nil {
		logAction(traceID, actionConfigProviderDelete, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderDelete, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionConfigProviderDelete, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeSetActiveProviderAction(req setActiveProviderRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigProviderSetActive, "running", nil)
	if err := s.configStore.SetActiveProvider(req.Name); err != nil {
		logAction(traceID, actionConfigProviderSetActive, "error", err)
		return nil, configProviderStatusCode(err), err
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderSetActive, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionConfigProviderSetActive, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) providerListPayload() (providerListResponse, error) {
	providers, err := s.configStore.ListProviders()
	if err != nil {
		return providerListResponse{}, err
	}
	return providerListResponse{
		Providers:      buildProviderConfigResponses(providers),
		ActiveProvider: s.configStore.Snapshot().Provider,
	}, nil
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
			ModelContextWindowTokens:   cloneModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: cloneModelTokenOverrides(provider.ModelResponseReserveTokens),
		})
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
