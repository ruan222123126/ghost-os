package config

import (
	"errors"
	"net/http"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
)

const (
	ActionProvidersGet      = "CONFIG_PROVIDERS_GET"
	ActionProviderCreate    = "CONFIG_PROVIDER_CREATE"
	ActionProviderUpdate    = "CONFIG_PROVIDER_UPDATE"
	ActionProviderDelete    = "CONFIG_PROVIDER_DELETE"
	ActionProviderSetActive = "CONFIG_PROVIDER_SET_ACTIVE"
)

type Store interface {
	ListProviders() ([]bridgeconfig.ProviderRecord, error)
	AddProvider(bridgeconfig.ProviderRecord) error
	UpdateProvider(string, bridgeconfig.ProviderRecord) error
	DeleteProvider(string) error
	SetActiveProvider(string) error
	Snapshot() bridgeconfig.Snapshot
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type Service struct {
	Store  Store
	Logger Logger
}

func (s Service) List(traceID string) (api.ProviderListResponse, int, error) {
	payload, err := s.providerListPayload()
	if err != nil {
		s.log(traceID, ActionProvidersGet, "error", err)
		return api.ProviderListResponse{}, http.StatusInternalServerError, err
	}
	s.log(traceID, ActionProvidersGet, "success", nil)
	return payload, http.StatusOK, nil
}

func (s Service) Create(req api.ProviderConfigInput, traceID string) (api.ProviderListResponse, int, error) {
	s.log(traceID, ActionProviderCreate, "running", nil)
	if err := s.Store.AddProvider(providerRecordFromInput(req)); err != nil {
		s.log(traceID, ActionProviderCreate, "error", err)
		return api.ProviderListResponse{}, MapProviderError(err), err
	}
	return s.updatedProviderList(traceID, ActionProviderCreate)
}

func (s Service) Update(
	name string,
	req api.ProviderConfigInput,
	traceID string,
) (api.ProviderListResponse, int, error) {
	s.log(traceID, ActionProviderUpdate, "running", nil)
	if err := s.Store.UpdateProvider(name, providerRecordFromInput(req)); err != nil {
		s.log(traceID, ActionProviderUpdate, "error", err)
		return api.ProviderListResponse{}, MapProviderError(err), err
	}
	return s.updatedProviderList(traceID, ActionProviderUpdate)
}

func (s Service) Delete(name string, traceID string) (api.ProviderListResponse, int, error) {
	s.log(traceID, ActionProviderDelete, "running", nil)
	if err := s.Store.DeleteProvider(name); err != nil {
		s.log(traceID, ActionProviderDelete, "error", err)
		return api.ProviderListResponse{}, MapProviderError(err), err
	}
	return s.updatedProviderList(traceID, ActionProviderDelete)
}

func (s Service) SetActive(req api.SetActiveProviderRequest, traceID string) (api.ProviderListResponse, int, error) {
	s.log(traceID, ActionProviderSetActive, "running", nil)
	if err := s.Store.SetActiveProvider(req.Name); err != nil {
		s.log(traceID, ActionProviderSetActive, "error", err)
		return api.ProviderListResponse{}, MapProviderError(err), err
	}
	return s.updatedProviderList(traceID, ActionProviderSetActive)
}

func (s Service) updatedProviderList(
	traceID string,
	action string,
) (api.ProviderListResponse, int, error) {
	payload, err := s.providerListPayload()
	if err != nil {
		s.log(traceID, action, "error", err)
		return api.ProviderListResponse{}, http.StatusInternalServerError, err
	}
	s.log(traceID, action, "success", nil)
	return payload, http.StatusOK, nil
}

func (s Service) providerListPayload() (api.ProviderListResponse, error) {
	providers, err := s.Store.ListProviders()
	if err != nil {
		return api.ProviderListResponse{}, err
	}
	return api.ProviderListResponse{
		Providers:      BuildProviderConfigResponses(providers),
		ActiveProvider: s.Store.Snapshot().Provider,
	}, nil
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}

func providerRecordFromInput(req api.ProviderConfigInput) bridgeconfig.ProviderRecord {
	return bridgeconfig.ProviderRecord{
		Name:                       req.Name,
		Type:                       llm.Provider(req.Type),
		BaseURL:                    StringValue(req.BaseURL),
		APIKey:                     CloneOptionalStringPointer(req.APIKey),
		Models:                     req.Models,
		ContextWindowTokens:        req.ContextWindowTokens,
		ResponseReserveTokens:      req.ResponseReserveTokens,
		ModelContextWindowTokens:   CloneModelTokenOverrides(req.ModelContextWindowTokens),
		ModelResponseReserveTokens: CloneModelTokenOverrides(req.ModelResponseReserveTokens),
	}
}

func BuildProviderConfigResponses(providers []bridgeconfig.ProviderRecord) []api.ProviderConfigResponse {
	if len(providers) == 0 {
		return []api.ProviderConfigResponse{}
	}

	out := make([]api.ProviderConfigResponse, 0, len(providers))
	for _, provider := range providers {
		out = append(out, api.ProviderConfigResponse{
			Name:                       provider.Name,
			Type:                       string(provider.Type),
			BaseURL:                    provider.BaseURL,
			Models:                     append([]string(nil), provider.Models...),
			APIKeySet:                  strings.TrimSpace(StringValue(provider.APIKey)) != "",
			ContextWindowTokens:        provider.ContextWindowTokens,
			ResponseReserveTokens:      provider.ResponseReserveTokens,
			ModelContextWindowTokens:   CloneModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: CloneModelTokenOverrides(provider.ModelResponseReserveTokens),
		})
	}
	return out
}

func MapProviderError(err error) int {
	switch {
	case errors.Is(err, bridgeconfig.ErrProviderNotFound):
		return http.StatusNotFound
	case errors.Is(err, bridgeconfig.ErrProviderExists):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func CloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func StringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func CloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}
