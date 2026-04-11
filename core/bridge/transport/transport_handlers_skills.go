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
	payload, code, err := t.service.ExecuteSkillListAction(traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleSkillByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}
	id, err := parseSkillPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	payload, code, err := t.service.ExecuteSkillDeleteAction(bridgeskills.SkillIDParams{ID: id}, traceID)
	respondServiceResult(w, traceID, payload, code, err)
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
