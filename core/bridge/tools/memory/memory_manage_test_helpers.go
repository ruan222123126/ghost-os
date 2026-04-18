package memory

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/memorystore"
)

func newMemoryTool(t *testing.T) *MemoryManageTool {
	t.Helper()
	store, err := memorystore.NewStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	tool := NewMemoryManageTool(store)
	t.Cleanup(func() {
		_ = tool.Close()
	})
	return tool
}

func executeMemoryTool(tool Tool, args any) (string, error) {
	payload, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	return tool.Execute(context.Background(), payload, "trace-test")
}

func mustExecuteMemoryTool(t *testing.T, tool Tool, args any) string {
	t.Helper()
	out, err := executeMemoryTool(tool, args)
	if err != nil {
		t.Fatalf("execute memory tool: %v", err)
	}
	return out
}

func decodeMemoryRecord(t *testing.T, output string) memorystore.Record {
	t.Helper()
	var record memorystore.Record
	if err := json.Unmarshal([]byte(output), &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	return record
}

func splitLines(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "\n")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}

func assertSystemRecordPage(t *testing.T, metadata map[string]any, total int, limit int, offset int, returned int) {
	t.Helper()
	assertMetadataInt(t, metadata, "total", total)
	assertMetadataInt(t, metadata, "limit", limit)
	assertMetadataInt(t, metadata, "offset", offset)
	assertMetadataInt(t, metadata, "returned", returned)
}

func assertMetadataInt(t *testing.T, metadata map[string]any, key string, want int) {
	t.Helper()
	if metadata == nil {
		t.Fatalf("expected metadata for %s", key)
	}
	value, ok := metadata[key]
	if !ok {
		t.Fatalf("expected metadata key %q", key)
	}
	got, ok := asInt(value)
	if !ok {
		t.Fatalf("metadata %q is not numeric: %#v", key, value)
	}
	if got != want {
		t.Fatalf("unexpected metadata %q: got=%d want=%d", key, got, want)
	}
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}
