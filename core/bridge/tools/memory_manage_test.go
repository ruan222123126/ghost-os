package tools

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestMemoryManageCreateReadUpdateDelete(t *testing.T) {
	tool := newMemoryTool(t)

	createOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://alpha",
		"content":   "hello",
		"metadata":  map[string]any{"tag": "first"},
	})
	created := decodeMemoryRecord(t, createOut)
	if created.URI != "user://alpha" {
		t.Fatalf("expected uri user://alpha, got %q", created.URI)
	}
	if created.Content != "hello" {
		t.Fatalf("expected content hello, got %q", created.Content)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be set: %+v", created)
	}
	if !reflect.DeepEqual(created.Metadata, explicitTestMetadata(map[string]any{"tag": "first"})) {
		t.Fatalf("unexpected metadata: %#v", created.Metadata)
	}

	readOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "read",
		"uri":       "user://alpha",
	})
	read := decodeMemoryRecord(t, readOut)
	if read.Content != "hello" {
		t.Fatalf("expected content hello, got %q", read.Content)
	}

	time.Sleep(2 * time.Millisecond)
	updateOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "update",
		"uri":       "user://alpha",
		"content":   "updated",
		"metadata":  map[string]any{"tag": "second"},
	})
	updated := decodeMemoryRecord(t, updateOut)
	if updated.Content != "updated" {
		t.Fatalf("expected updated content, got %q", updated.Content)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected created_at to remain unchanged")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("expected updated_at to advance")
	}
	if !reflect.DeepEqual(updated.Metadata, explicitTestMetadata(map[string]any{"tag": "second"})) {
		t.Fatalf("expected updated metadata, got %#v", updated.Metadata)
	}

	readOut = mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "read",
		"uri":       "user://alpha",
	})
	read = decodeMemoryRecord(t, readOut)
	if read.Content != "updated" {
		t.Fatalf("expected updated content, got %q", read.Content)
	}

	deleteOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "delete",
		"uri":       "user://alpha",
	})
	var deleted memoryDeleteResult
	if err := json.Unmarshal([]byte(deleteOut), &deleted); err != nil {
		t.Fatalf("decode delete result: %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("expected deleted=true, got false")
	}

	if _, err := executeMemoryTool(tool, map[string]any{"operation": "read", "uri": "user://alpha"}); err == nil {
		t.Fatalf("expected read after delete to fail")
	}
}

func TestMemoryManageErrors(t *testing.T) {
	tool := newMemoryTool(t)

	_, err := executeMemoryTool(tool, map[string]any{
		"operation": "create",
		"uri":       "user://dup",
		"content":   "hello",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if _, err := executeMemoryTool(tool, map[string]any{
		"operation": "create",
		"uri":       "user://dup",
		"content":   "hello",
	}); err == nil {
		t.Fatalf("expected duplicate create to fail")
	}
	if _, err := executeMemoryTool(tool, map[string]any{
		"operation": "update",
		"uri":       "user://missing",
		"content":   "update",
	}); err == nil {
		t.Fatalf("expected update missing to fail")
	}
	if _, err := executeMemoryTool(tool, map[string]any{
		"operation": "read",
		"uri":       "user://missing",
	}); err == nil {
		t.Fatalf("expected read missing to fail")
	}
	if _, err := executeMemoryTool(tool, map[string]any{
		"operation": "create",
		"uri":       "user://bad\nnext",
		"content":   "hello",
	}); err == nil {
		t.Fatalf("expected invalid uri to fail")
	}
}

func TestMemoryManageListAndSearch(t *testing.T) {
	tool := newMemoryTool(t)

	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://alpha",
		"content":   "hello world",
	})
	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://beta",
		"content":   "hello there",
	})
	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "project://gamma",
		"content":   "other",
	})

	listOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "list",
		"prefix":    "user://",
	})
	var listResult memoryListResult
	if err := json.Unmarshal([]byte(listOut), &listResult); err != nil {
		t.Fatalf("decode list result: %v", err)
	}
	if listResult.Total != 2 {
		t.Fatalf("expected total 2, got %d", listResult.Total)
	}
	if len(listResult.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(listResult.Items))
	}
	uris := []string{listResult.Items[0].URI, listResult.Items[1].URI}
	sorted := append([]string(nil), uris...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(uris, sorted) {
		t.Fatalf("expected list to be sorted by uri: %v", uris)
	}

	listOut = mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "list",
		"prefix":    "user://a",
	})
	if err := json.Unmarshal([]byte(listOut), &listResult); err != nil {
		t.Fatalf("decode list result: %v", err)
	}
	if listResult.Total != 1 || len(listResult.Items) != 1 || listResult.Items[0].URI != "user://alpha" {
		t.Fatalf("unexpected prefix list result: %+v", listResult)
	}

	searchOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "search",
		"query":     "world",
	})
	if err := json.Unmarshal([]byte(searchOut), &listResult); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if listResult.Total != 1 || len(listResult.Items) != 1 || listResult.Items[0].URI != "user://alpha" {
		t.Fatalf("unexpected search result: %+v", listResult)
	}
}

func TestMemoryManageUpdatePreservesMetadataWhenOmitted(t *testing.T) {
	tool := newMemoryTool(t)

	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://alpha",
		"content":   "first",
		"metadata":  map[string]any{"tag": "persist"},
	})

	updateOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "update",
		"uri":       "user://alpha",
		"content":   "second",
	})
	updated := decodeMemoryRecord(t, updateOut)
	if !reflect.DeepEqual(updated.Metadata, explicitTestMetadata(map[string]any{"tag": "persist"})) {
		t.Fatalf("expected metadata to be preserved, got %#v", updated.Metadata)
	}
}

func explicitTestMetadata(extra map[string]any) map[string]any {
	metadata := map[string]any{
		"source_kind": "explicit",
		"scope_type":  "user",
		"scope_id":    "local-user",
	}
	for key, value := range extra {
		metadata[key] = value
	}
	return metadata
}
