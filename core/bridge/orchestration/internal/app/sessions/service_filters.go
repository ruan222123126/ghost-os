package sessions

import (
	"strings"

	"ghost-os/bridge/session"
)

func filterHiddenSessionMetadata(
	metadata []session.SessionMetadata,
	hiddenSessionIDs map[string]struct{},
) []session.SessionMetadata {
	if len(metadata) == 0 || len(hiddenSessionIDs) == 0 {
		return metadata
	}
	filtered := make([]session.SessionMetadata, 0, len(metadata))
	for _, item := range metadata {
		if _, hidden := hiddenSessionIDs[item.ID]; hidden {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func limitSessionMetadata(metadata []session.SessionMetadata, limit int) []session.SessionMetadata {
	if limit <= 0 || len(metadata) <= limit {
		return metadata
	}
	return metadata[:limit]
}

func stringSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	return set
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}
