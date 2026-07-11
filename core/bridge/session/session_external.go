package session

import (
	"strings"
	"time"
)

type ExternalPendingApproval struct {
	ID        string         `json:"id"`
	Provider  string         `json:"provider"`
	Kind      string         `json:"kind"`
	Tool      string         `json:"tool,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	Prompt    string         `json:"prompt,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

type ExternalRuntime struct {
	Provider         string                    `json:"provider"`
	Status           string                    `json:"status"`
	ThreadID         string                    `json:"thread_id,omitempty"`
	TurnID           string                    `json:"turn_id,omitempty"`
	PermissionMode   string                    `json:"permission_mode,omitempty"`
	Model            string                    `json:"model,omitempty"`
	Effort           string                    `json:"effort,omitempty"`
	CWD              string                    `json:"cwd,omitempty"`
	ProjectRoot      string                    `json:"project_root,omitempty"`
	PendingApprovals []ExternalPendingApproval `json:"pending_approvals,omitempty"`
	StartedAt        time.Time                 `json:"started_at,omitempty"`
	UpdatedAt        time.Time                 `json:"updated_at,omitempty"`
}

func (s *Session) SetExternalRuntime(runtime *ExternalRuntime) {
	if s == nil {
		return
	}
	s.ExternalRuntime = cloneExternalRuntime(runtime)
	if s.ExternalRuntime != nil {
		s.UpdatedAt = s.ExternalRuntime.UpdatedAt
		if s.UpdatedAt.IsZero() {
			s.UpdatedAt = time.Now().UTC()
		}
	}
}

func cloneExternalRuntime(raw *ExternalRuntime) *ExternalRuntime {
	if raw == nil {
		return nil
	}
	cloned := *raw
	cloned.Provider = strings.TrimSpace(raw.Provider)
	cloned.Status = strings.TrimSpace(raw.Status)
	cloned.ThreadID = strings.TrimSpace(raw.ThreadID)
	cloned.TurnID = strings.TrimSpace(raw.TurnID)
	cloned.PermissionMode = strings.TrimSpace(raw.PermissionMode)
	cloned.Model = strings.TrimSpace(raw.Model)
	cloned.Effort = strings.TrimSpace(raw.Effort)
	cloned.CWD = strings.TrimSpace(raw.CWD)
	cloned.ProjectRoot = strings.TrimSpace(raw.ProjectRoot)
	cloned.PendingApprovals = cloneExternalPendingApprovals(raw.PendingApprovals)
	return &cloned
}

func cloneExternalPendingApprovals(raw []ExternalPendingApproval) []ExternalPendingApproval {
	if len(raw) == 0 {
		return nil
	}
	out := make([]ExternalPendingApproval, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		out = append(out, ExternalPendingApproval{
			ID:        id,
			Provider:  strings.TrimSpace(item.Provider),
			Kind:      strings.TrimSpace(item.Kind),
			Tool:      strings.TrimSpace(item.Tool),
			CallID:    strings.TrimSpace(item.CallID),
			Prompt:    strings.TrimSpace(item.Prompt),
			Payload:   cloneAnyMap(item.Payload),
			CreatedAt: item.CreatedAt,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
