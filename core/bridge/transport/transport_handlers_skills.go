package transport

import (
	"errors"
	"net/http"
	"strings"

	bridgeskills "ghost-os/bridge/skills"
)

func (t *transport) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	traceID := resolveTraceID("", r)
	result, err := t.service.ExecuteSkillListAction(traceID)
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
	var req bridgeskills.SkillUpdateRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, callErr := t.service.ExecuteSkillUpdateAction(bridgeskills.SkillIDParams{ID: id}, req, traceID)
	respondServiceContractResult(w, traceID, result, callErr)
}

func (t *transport) handleSkillDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSkillPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	result, callErr := t.service.ExecuteSkillDeleteAction(bridgeskills.SkillIDParams{ID: id}, traceID)
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
