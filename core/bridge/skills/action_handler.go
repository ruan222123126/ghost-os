package skills

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

// Action constants shared with transport layer.
const (
	ActionSkillList   = "SKILL_LIST"
	ActionSkillDelete = "SKILL_DELETE"
)

// Skill ID encoding constants.
const (
	SkillIDPrefix    = "skill_"
	SkillIDSeparator = "|"

	skillSourceRepo = "repo"
	skillSourceUser = "user"
)

// Skill source identifiers exported for test usage.
const (
	SkillSourceRepo = skillSourceRepo
	SkillSourceUser = skillSourceUser
)

var (
	ErrInvalidSkillID      = errors.New("invalid skill id")
	ErrSkillNotFound       = errors.New("skill not found")
	ErrSkillPathForbidden  = errors.New("skill path is outside managed roots")
	ErrSkillSourceNotFound = errors.New("skill source root is not configured")
)

// API param types decoded from JSON request bodies.
type SkillIDParams struct {
	ID string `json:"id"`
}

type SkillPayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Source      string `json:"source"`
}

type SkillDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// LogFunc logs action lifecycle events.
type LogFunc func(traceID string, action string, status string, err error)

// Store provides access to project configuration for skill discovery.
type Store interface {
	Config() (Config, error)
}

// Config describes the minimal config subset needed for skill discovery.
type Config struct {
	ProjectRoot string
}

// ActionHandler implements all skill action use cases.
type ActionHandler struct {
	store Store
	log   LogFunc
}

// NewActionHandler creates a skill action handler with the given config store.
func NewActionHandler(store Store, log LogFunc) *ActionHandler {
	return &ActionHandler{store: store, log: log}
}

// ExecuteListAction handles the SKILL_LIST action.
func (h *ActionHandler) ExecuteListAction(traceID string) (any, int, error) {
	items, _, err := h.discoverManagedSkills()
	if err != nil {
		h.logAction(traceID, ActionSkillList, "error", err)
		return nil, mapSkillError(err), err
	}
	payload := make([]SkillPayload, 0, len(items))
	for _, item := range items {
		payload = append(payload, toSkillPayload(item))
	}
	h.logAction(traceID, ActionSkillList, "success", nil)
	return payload, http.StatusOK, nil
}

// ExecuteDeleteAction handles the SKILL_DELETE action.
func (h *ActionHandler) ExecuteDeleteAction(params SkillIDParams, traceID string) (any, int, error) {
	decoded, err := DecodeSkillID(params.ID)
	if err != nil {
		h.logAction(traceID, ActionSkillDelete, "error", err)
		return nil, mapSkillError(err), err
	}
	items, roots, err := h.discoverManagedSkills()
	if err != nil {
		h.logAction(traceID, ActionSkillDelete, "error", err)
		return nil, mapSkillError(err), err
	}
	item, err := findManagedSkill(items, decoded, strings.TrimSpace(params.ID))
	if err != nil {
		h.logAction(traceID, ActionSkillDelete, "error", err)
		return nil, mapSkillError(err), err
	}
	if err := removeManagedSkill(item); err != nil {
		h.logAction(traceID, ActionSkillDelete, "error", err)
		return nil, mapSkillError(err), err
	}
	if err := refreshManagedSkillCatalog(roots); err != nil {
		h.logAction(traceID, ActionSkillDelete, "error", err)
		return nil, mapSkillError(err), err
	}
	h.logAction(traceID, ActionSkillDelete, "success", nil)
	return SkillDeleteResponse{ID: item.ID, Deleted: true}, http.StatusOK, nil
}

// ManagedSkillFromDiscovery converts a raw Skill discovery result to a managedSkill.
// Exported for test usage.
func ManagedSkillFromDiscovery(source string, root string, item Skill) (managedSkill, error) {
	return managedSkillFromDiscovery(source, root, item)
}

func (h *ActionHandler) logAction(traceID, action, status string, err error) {
	if h.log != nil {
		h.log(traceID, action, status, err)
	}
}

// Internal types.

type skillRoots struct {
	Repo string
	User string
}

type managedSkill struct {
	ID           string
	Name         string
	Description  string
	Path         string
	Source       string
	Root         string
	RelativePath string
}

type decodedSkillID struct {
	Source       string
	RelativePath string
}

func mapSkillError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidSkillID), errors.Is(err, ErrSkillPathForbidden), errors.Is(err, ErrSkillSourceNotFound):
		return http.StatusBadRequest
	case errors.Is(err, ErrSkillNotFound), os.IsNotExist(err):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func toSkillPayload(item managedSkill) SkillPayload {
	return SkillPayload{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Path:        item.Path,
		Source:      item.Source,
	}
}
