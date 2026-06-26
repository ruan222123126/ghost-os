package sessions

import (
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
)

var ErrSessionAppendMessagesRequired = errors.New("session append messages are required")
var ErrSessionAppendRoleInvalid = errors.New("session append message role must be user or assistant")
var ErrSessionAppendTextRequired = errors.New("session append message text is required")

func AppendSession(store Store, req api.SessionAppendRequest) (api.SessionAppendResponse, error) {
	if store == nil {
		return api.SessionAppendResponse{}, ErrSessionStoreRequired
	}

	sessionID, err := RequireSessionID(req.SessionID)
	if err != nil {
		return api.SessionAppendResponse{}, err
	}
	if len(req.Messages) == 0 {
		return api.SessionAppendResponse{}, ErrSessionAppendMessagesRequired
	}

	loaded, err := store.Load(sessionID)
	switch {
	case err == nil:
	case errors.Is(err, session.ErrSessionNotFound):
		loaded = session.NewSession("")
		loaded.ID = sessionID
	default:
		return api.SessionAppendResponse{}, err
	}

	if req.ExpectedHead != nil && loaded.MessageCount != *req.ExpectedHead {
		return api.SessionAppendResponse{
			SessionID:    sessionID,
			Status:       "conflict",
			MessageCount: loaded.MessageCount,
			UpdatedAt:    loaded.UpdatedAt.UTC().Format(time.RFC3339),
		}, nil
	}

	if title := strings.TrimSpace(req.Title); title != "" && strings.TrimSpace(loaded.Title) == "" {
		loaded.Title = title
	}
	loaded.EndedAt = time.Time{}

	for _, message := range req.Messages {
		normalized, err := normalizeAppendMessage(message)
		if err != nil {
			return api.SessionAppendResponse{}, err
		}
		loaded.AddMessage(normalized)
		if strings.TrimSpace(loaded.Title) == "" && normalized.Role == llm.RoleUser {
			loaded.Title = strings.TrimSpace(normalized.Text)
		}
	}

	if err := store.Save(loaded); err != nil {
		return api.SessionAppendResponse{}, err
	}
	return api.SessionAppendResponse{
		SessionID:    loaded.ID,
		Status:       "appended",
		MessageCount: loaded.MessageCount,
		UpdatedAt:    loaded.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func normalizeAppendMessage(message api.SessionAppendMessage) (llm.Message, error) {
	text := strings.TrimSpace(message.Text)
	if text == "" {
		return llm.Message{}, ErrSessionAppendTextRequired
	}

	switch strings.ToLower(strings.TrimSpace(message.Role)) {
	case string(llm.RoleUser):
		return llm.Message{Role: llm.RoleUser, Text: text}, nil
	case string(llm.RoleAssistant):
		return llm.Message{Role: llm.RoleAssistant, Text: text}, nil
	default:
		return llm.Message{}, ErrSessionAppendRoleInvalid
	}
}
