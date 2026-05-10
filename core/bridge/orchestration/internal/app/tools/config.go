package tools

import (
	"errors"
	"net/http"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
)

const (
	ActionList   = "TOOL_LIST"
	ActionUpdate = "TOOL_UPDATE"
)

type Store interface {
	ListTools() ([]bridgeconfig.ToolRecord, error)
	UpdateTool(bridgeconfig.ToolUpdateRequest) error
}

type SchemaProvider interface {
	ListToolInputSchemas() (map[string]map[string]any, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type Service struct {
	Store          Store
	SchemaProvider SchemaProvider
	Logger         Logger
}

func (s Service) List(traceID string) (any, int, error) {
	items, err := s.Store.ListTools()
	if err != nil {
		s.log(traceID, ActionList, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	schemasByName, err := s.SchemaProvider.ListToolInputSchemas()
	if err != nil {
		s.log(traceID, ActionList, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	s.log(traceID, ActionList, "success", nil)
	return BuildPayloads(items, schemasByName), http.StatusOK, nil
}

func (s Service) Update(
	params api.ToolNameParams,
	req api.ToolUpdateRequest,
	traceID string,
) (any, int, error) {
	if err := s.Store.UpdateTool(UpdateRequestToStore(params.Name, req)); err != nil {
		s.log(traceID, ActionUpdate, "error", err)
		return nil, MapConfigError(err), err
	}
	item, err := s.updatedTool(params.Name)
	if err != nil {
		s.log(traceID, ActionUpdate, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	s.log(traceID, ActionUpdate, "success", nil)
	return PayloadFromRecord(item, nil), http.StatusOK, nil
}

func (s Service) updatedTool(name string) (bridgeconfig.ToolRecord, error) {
	items, err := s.Store.ListTools()
	if err != nil {
		return bridgeconfig.ToolRecord{}, err
	}
	trimmed := strings.TrimSpace(name)
	for _, item := range items {
		if item.Name == trimmed {
			return item, nil
		}
	}
	return bridgeconfig.ToolRecord{}, errors.New("tool not found after update")
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}

func BuildPayloads(
	items []bridgeconfig.ToolRecord,
	schemasByName map[string]map[string]any,
) []api.ToolPayload {
	payload := make([]api.ToolPayload, 0, len(items))
	for _, item := range items {
		payload = append(payload, PayloadFromRecord(item, SchemaByToolName(schemasByName, item.Name)))
	}
	return payload
}

func PayloadFromRecord(item bridgeconfig.ToolRecord, inputSchema map[string]any) api.ToolPayload {
	return api.ToolPayload{
		Name:            item.Name,
		Enabled:         item.Enabled,
		PromptOverride:  strings.TrimSpace(item.PromptOverride),
		SandboxMemoryMB: CloneOptionalInt(item.SandboxMemoryMB),
		InputSchema:     inputSchema,
	}
}

func UpdateRequestToStore(name string, req api.ToolUpdateRequest) bridgeconfig.ToolUpdateRequest {
	return bridgeconfig.ToolUpdateRequest{
		Name:            strings.TrimSpace(name),
		Enabled:         req.Enabled,
		PromptOverride:  req.PromptOverride,
		SandboxMemoryMB: req.SandboxMemoryMB,
	}
}

func MapConfigError(err error) int {
	switch {
	case errors.Is(err, bridgeconfig.ErrToolNameRequired),
		errors.Is(err, bridgeconfig.ErrToolUpdateEmpty),
		errors.Is(err, bridgeconfig.ErrToolConfigInvalid):
		return http.StatusBadRequest
	case errors.Is(err, bridgeconfig.ErrToolNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func CloneOptionalInt(raw *int) *int {
	if raw == nil {
		return nil
	}
	value := *raw
	return &value
}
