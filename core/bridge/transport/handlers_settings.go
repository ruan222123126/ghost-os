// bridgeconfig.Config and provider HTTP handlers.

package transport

import (
	"errors"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"net/url"
	"strings"
)

func (t *transport) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.config.Get(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.ConfigUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.config.Update(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleConfigProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.config.ListProviders(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.ProviderCreateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.config.CreateProvider(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleConfigProviderExport(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.ProviderExportRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.usecases.config.ExportProvider(req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleConfigProviderByName(w http.ResponseWriter, r *http.Request) {
	name, ok := providerNameFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "provider name is required", "")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req bridgeorchestration.ProviderUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.config.UpdateProvider(name, req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodDelete:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.config.DeleteProvider(name, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleActiveProvider(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	var req bridgeorchestration.SetActiveProviderRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.usecases.config.SetActiveProvider(req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func providerNameFromPath(path string) (string, bool) {
	rawName := strings.TrimSpace(strings.TrimPrefix(path, "/api/config/providers/"))
	if rawName == "" || strings.Contains(rawName, "/") {
		return "", false
	}
	decoded, err := url.PathUnescape(rawName)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", false
	}
	return decoded, true
}

func (t *transport) handleSystemPrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.config.GetSystemPrompts(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPatch:
		var req bridgeorchestration.SystemPromptUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.config.UpdateSystemPrompts(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handlePresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.presets.List(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.PresetCreateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.presets.Create(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handlePresetByID(w http.ResponseWriter, r *http.Request) {
	if presetID, ok := presetActivationIDFromPath(r.URL.Path); ok {
		handlePresetActivationRequest(t, w, r, presetID)
		return
	}

	presetID, err := presetIDFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req bridgeorchestration.PresetUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, callErr := t.usecases.presets.Update(presetID, req, traceID)
		respondServiceContractResult(w, traceID, result, callErr)
	case http.MethodDelete:
		traceID := resolveTraceID("", r)
		result, callErr := t.usecases.presets.Delete(presetID, traceID)
		respondServiceContractResult(w, traceID, result, callErr)
	default:
		writeMethodNotAllowed(w)
	}
}

func handlePresetActivationRequest(
	t *transport,
	w http.ResponseWriter,
	r *http.Request,
	presetID string,
) {
	if r.Method != http.MethodPut {
		writeMethodNotAllowed(w)
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.usecases.presets.Apply(presetID, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func presetIDFromPath(rawPath string) (string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/presets/"))
	if path == "" {
		return "", errors.New("preset id is required")
	}
	segments := strings.Split(path, "/")
	if len(segments) != 1 || strings.TrimSpace(segments[0]) == "" {
		return "", errors.New("invalid preset path")
	}
	decoded, err := url.PathUnescape(strings.TrimSpace(segments[0]))
	if err != nil {
		return "", errors.New("invalid preset path")
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", errors.New("invalid preset path")
	}
	return decoded, nil
}

func presetActivationIDFromPath(rawPath string) (string, bool) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/presets/"))
	segments := strings.Split(path, "/")
	if len(segments) != 2 || strings.TrimSpace(segments[1]) != "activate" {
		return "", false
	}

	decoded, err := url.PathUnescape(strings.TrimSpace(segments[0]))
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", false
	}
	return decoded, true
}

func (t *transport) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	traceID := resolveTraceID("", r)
	result, err := t.usecases.skills.List(traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleSkillByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPatch:
		t.handleSkillPatch(w, r)
	case http.MethodDelete:
		t.handleSkillDelete(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleSkillPatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseSkillPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	var req bridgeorchestration.SkillUpdateRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, callErr := t.usecases.skills.Update(bridgeorchestration.SkillIDParams{ID: id}, req, traceID)
	respondServiceContractResult(w, traceID, result, callErr)
}

func (t *transport) handleSkillDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSkillPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	result, callErr := t.usecases.skills.Delete(bridgeorchestration.SkillIDParams{ID: id}, traceID)
	respondServiceContractResult(w, traceID, result, callErr)
}

func parseSkillPath(rawPath string) (string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/skills/"))
	if path == "" {
		return "", errors.New("skill id is required")
	}
	segments := strings.Split(path, "/")
	if len(segments) != 1 || strings.TrimSpace(segments[0]) == "" {
		return "", errors.New("invalid skill path")
	}
	return strings.TrimSpace(segments[0]), nil
}
