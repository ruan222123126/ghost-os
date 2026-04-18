package orchestration

import (
	"errors"
	"net/http"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

const (
	busActionToolList   = "TOOL_LIST"
	busActionToolUpdate = "TOOL_UPDATE"
)

func (s *bridgeService) executeToolListAction(traceID string) (any, int, error) {
	items, err := s.configStore.ListTools()
	if err != nil {
		logAction(traceID, busActionToolList, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	schemasByName, err := s.listToolInputSchemas()
	if err != nil {
		logAction(traceID, busActionToolList, "error", err)
		return nil, http.StatusInternalServerError, err
	}

	payload := make([]toolPayload, 0, len(items))
	for _, item := range items {
		entry := toolPayload{
			Name:           item.Name,
			Enabled:        item.Enabled,
			PromptOverride: strings.TrimSpace(item.PromptOverride),
		}
		if inputSchema, ok := schemaByToolName(schemasByName, item.Name); ok {
			entry.InputSchema = inputSchema
		}
		payload = append(payload, entry)
	}
	logAction(traceID, busActionToolList, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeToolUpdateAction(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (any, int, error) {
	if err := s.configStore.UpdateTool(toolUpdateRequestToStoreRequest(params.Name, req)); err != nil {
		logAction(traceID, busActionToolUpdate, "error", err)
		return nil, mapToolConfigError(err), err
	}

	items, err := s.configStore.ListTools()
	if err != nil {
		logAction(traceID, busActionToolUpdate, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	name := strings.TrimSpace(params.Name)
	for _, item := range items {
		if item.Name != name {
			continue
		}
		logAction(traceID, busActionToolUpdate, "success", nil)
		return toolPayload{
			Name:           item.Name,
			Enabled:        item.Enabled,
			PromptOverride: strings.TrimSpace(item.PromptOverride),
		}, http.StatusOK, nil
	}

	missingErr := errors.New("tool not found after update")
	logAction(traceID, busActionToolUpdate, "error", missingErr)
	return nil, http.StatusInternalServerError, missingErr
}

func mapToolConfigError(err error) int {
	switch {
	case errors.Is(err, bridgeconfig.ErrToolNameRequired),
		errors.Is(err, bridgeconfig.ErrToolUpdateEmpty):
		return http.StatusBadRequest
	case errors.Is(err, bridgeconfig.ErrToolNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
