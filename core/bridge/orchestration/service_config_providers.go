package orchestration

import (
	"errors"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
)

const (
	actionConfigProvidersGet      = "CONFIG_PROVIDERS_GET"
	actionConfigProviderCreate    = "CONFIG_PROVIDER_CREATE"
	actionConfigProviderUpdate    = "CONFIG_PROVIDER_UPDATE"
	actionConfigProviderDelete    = "CONFIG_PROVIDER_DELETE"
	actionConfigProviderSetActive = "CONFIG_PROVIDER_SET_ACTIVE"
)

func (s *bridgeService) executeProvidersGetAction(traceID string) (ServiceResult, error) {
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProvidersGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, actionConfigProvidersGet, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (ServiceResult, error) {
	logAction(traceID, actionConfigProviderCreate, "running", nil)
	if err := s.configStore.AddProvider(bridgeconfig.ProviderRecord{
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
		return ServiceResult{}, wrapServiceError(configProviderErrorKind(err), err)
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderCreate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, actionConfigProviderCreate, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) executeProviderUpdateAction(name string, req providerUpdateRequest, traceID string) (ServiceResult, error) {
	logAction(traceID, actionConfigProviderUpdate, "running", nil)
	if err := s.configStore.UpdateProvider(name, bridgeconfig.ProviderRecord{
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
		return ServiceResult{}, wrapServiceError(configProviderErrorKind(err), err)
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, actionConfigProviderUpdate, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (ServiceResult, error) {
	logAction(traceID, actionConfigProviderDelete, "running", nil)
	if err := s.configStore.DeleteProvider(name); err != nil {
		logAction(traceID, actionConfigProviderDelete, "error", err)
		return ServiceResult{}, wrapServiceError(configProviderErrorKind(err), err)
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderDelete, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, actionConfigProviderDelete, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) executeSetActiveProviderAction(req setActiveProviderRequest, traceID string) (ServiceResult, error) {
	logAction(traceID, actionConfigProviderSetActive, "running", nil)
	if err := s.configStore.SetActiveProvider(req.Name); err != nil {
		logAction(traceID, actionConfigProviderSetActive, "error", err)
		return ServiceResult{}, wrapServiceError(configProviderErrorKind(err), err)
	}
	payload, err := s.providerListPayload()
	if err != nil {
		logAction(traceID, actionConfigProviderSetActive, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, actionConfigProviderSetActive, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) providerListPayload() (providerListResponse, error) {
	providers, err := s.configStore.ListProviders()
	if err != nil {
		return providerListResponse{}, err
	}
	return providerListResponse{
		Providers:      buildProviderConfigResponses(providers),
		ActiveProvider: configResponseFromSnapshot(s.configStore.Snapshot()).Provider,
	}, nil
}

func buildProviderConfigResponses(providers []bridgeconfig.ProviderRecord) []providerConfigResponse {
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

func configProviderErrorKind(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, bridgeconfig.ErrProviderNotFound):
		return ServiceErrorNotFound
	case errors.Is(err, bridgeconfig.ErrProviderExists):
		return ServiceErrorConflict
	default:
		return ServiceErrorInvalidInput
	}
}
