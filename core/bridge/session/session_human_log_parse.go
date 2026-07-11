package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseSessionHumanLogDocument(raw string) (sessionHumanLogDocument, error) {
	meta, err := parseSessionHumanLogMeta(raw)
	if err != nil {
		return sessionHumanLogDocument{}, err
	}
	body, err := parseSessionHumanLogBody(raw)
	if err != nil {
		return sessionHumanLogDocument{}, err
	}
	return sessionHumanLogDocument{
		Meta: meta,
		Body: body,
	}, nil
}

func parseSessionHumanLogMeta(raw string) (sessionHumanLogMeta, error) {
	metaPayload, err := parseSessionHumanLogMetaPayload(raw)
	if err != nil {
		return sessionHumanLogMeta{}, err
	}

	var meta sessionHumanLogMeta
	if err := json.Unmarshal([]byte(metaPayload), &meta); err != nil {
		return sessionHumanLogMeta{}, fmt.Errorf("decode session human log meta: %w", err)
	}
	if err := validateSessionHumanLogMeta(meta); err != nil {
		return sessionHumanLogMeta{}, err
	}
	return meta, nil
}

func parseSessionHumanLogMetaPayload(raw string) (string, error) {
	start := strings.Index(raw, sessionHumanLogMetaPrefix)
	if start < 0 {
		return "", errorsSessionHumanLogMetaNotFound()
	}
	remaining := raw[start+len(sessionHumanLogMetaPrefix):]
	end := strings.Index(remaining, sessionHumanLogMetaSuffix)
	if end < 0 {
		return "", fmt.Errorf("decode session human log meta: missing suffix %q", strings.TrimSpace(sessionHumanLogMetaSuffix))
	}
	return remaining[:end], nil
}

func parseSessionHumanLogBody(raw string) (string, error) {
	start := strings.Index(raw, sessionHumanLogBodyMarker)
	if start < 0 {
		return "", fmt.Errorf("decode session human log body: missing marker %q", strings.TrimSpace(sessionHumanLogBodyMarker))
	}
	return raw[start+len(sessionHumanLogBodyMarker):], nil
}

func validateSessionHumanLogMeta(meta sessionHumanLogMeta) error {
	if strings.TrimSpace(meta.SessionID) == "" {
		return fmt.Errorf("decode session human log meta: session_id is empty")
	}
	if !isValidSessionHumanLogMode(meta.ExportMode) {
		return fmt.Errorf("decode session human log meta: unsupported export_mode %q", meta.ExportMode)
	}
	if meta.LastExportedIndex < -1 {
		return fmt.Errorf("decode session human log meta: invalid last_exported_index %d", meta.LastExportedIndex)
	}
	return nil
}

func errorsSessionHumanLogMetaNotFound() error {
	return fmt.Errorf("decode session human log meta: missing marker %q", strings.TrimSpace(sessionHumanLogMetaPrefix))
}

func isValidSessionHumanLogMode(mode sessionHumanLogMode) bool {
	switch mode {
	case sessionHumanLogModeSummary, sessionHumanLogModeFull:
		return true
	default:
		return false
	}
}
