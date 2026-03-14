package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"ghost-os/bridge/memorystore"
)

func decodeMemoryManageArgs(raw json.RawMessage, target *memoryManageArgs) error {
	return decodeJSONArgs(raw, target)
}

func decodeJSONArgs(raw json.RawMessage, target any) error {
	source := bytes.TrimSpace(raw)
	if len(source) == 0 || bytes.Equal(source, []byte("null")) {
		source = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid params: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func requireMemoryURI(value *string, operation string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("uri is required for %s", operation)
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "", fmt.Errorf("uri is required for %s", operation)
	}
	return trimmed, nil
}

func requireMemoryContent(value *string, operation string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("content is required for %s", operation)
	}
	if strings.TrimSpace(*value) == "" {
		return "", fmt.Errorf("content is required for %s", operation)
	}
	return *value, nil
}

func ensureWritableMemoryURI(uri string) error {
	switch uri {
	case systemIndexURI, systemRecentURI:
		return fmt.Errorf("%s is read-only", uri)
	default:
		return nil
	}
}

func resolveMemoryPaging(limit *int, offset *int) (int, int, error) {
	resolvedLimit := defaultMemoryLimit
	if limit != nil {
		resolvedLimit = *limit
	}
	resolvedOffset := defaultMemoryOffset
	if offset != nil {
		resolvedOffset = *offset
	}
	if resolvedLimit < 1 {
		return 0, 0, fmt.Errorf("limit must be >= 1")
	}
	if resolvedOffset < 0 {
		return 0, 0, fmt.Errorf("offset must be >= 0")
	}
	return resolvedLimit, resolvedOffset, nil
}

func buildSystemRecord(uri string, items []string, total int, limit int, offset int) memorystore.Record {
	now := time.Now().UTC()
	return memorystore.Record{
		URI:     uri,
		Content: strings.Join(items, "\n"),
		Metadata: map[string]any{
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"returned": len(items),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mapMemoryStoreError(err error, uri string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, memorystore.ErrAlreadyExists):
		return fmt.Errorf("memory %q already exists", uri)
	case errors.Is(err, memorystore.ErrNotFound):
		return fmt.Errorf("memory %q not found", uri)
	case errors.Is(err, memorystore.ErrInvalidURI):
		return err
	default:
		return err
	}
}

func marshalMemoryManageOutput(payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode memory result: %w", err)
	}
	return string(encoded), nil
}
