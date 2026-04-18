package memory

import (
	"reflect"
	"testing"
	"time"
)

func TestMemoryManageSystemIndexAndRecent(t *testing.T) {
	tool := newMemoryTool(t)

	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://alpha",
		"content":   "first",
	})
	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "create",
		"uri":       "user://beta",
		"content":   "second",
	})
	time.Sleep(2 * time.Millisecond)
	mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "update",
		"uri":       "user://alpha",
		"content":   "first-updated",
	})

	indexOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "read",
		"uri":       "system://index",
	})
	indexRecord := decodeMemoryRecord(t, indexOut)
	if got := splitLines(indexRecord.Content); !reflect.DeepEqual(got, []string{"user://alpha", "user://beta"}) {
		t.Fatalf("unexpected system index order: %v", got)
	}
	assertSystemRecordPage(t, indexRecord.Metadata, 2, defaultMemoryLimit, defaultMemoryOffset, 2)

	recentOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "read",
		"uri":       "system://recent",
	})
	recentRecord := decodeMemoryRecord(t, recentOut)
	if got := splitLines(recentRecord.Content); !reflect.DeepEqual(got, []string{"user://alpha", "user://beta"}) {
		t.Fatalf("unexpected system recent order: %v", got)
	}
	assertSystemRecordPage(t, recentRecord.Metadata, 2, defaultMemoryLimit, defaultMemoryOffset, 2)
}

func TestMemoryManageSystemReadSupportsPagingWithoutMetadataDuplication(t *testing.T) {
	tool := newMemoryTool(t)

	for _, uri := range []string{"user://alpha", "user://beta", "user://gamma"} {
		mustExecuteMemoryTool(t, tool, map[string]any{
			"operation": "create",
			"uri":       uri,
			"content":   uri,
		})
	}

	indexOut := mustExecuteMemoryTool(t, tool, map[string]any{
		"operation": "read",
		"uri":       "system://index",
		"limit":     1,
		"offset":    1,
	})
	indexRecord := decodeMemoryRecord(t, indexOut)
	if got := splitLines(indexRecord.Content); !reflect.DeepEqual(got, []string{"user://beta"}) {
		t.Fatalf("unexpected paged index content: %v", got)
	}
	assertSystemRecordPage(t, indexRecord.Metadata, 3, 1, 1, 1)
	if _, ok := indexRecord.Metadata["items"]; ok {
		t.Fatalf("system metadata should not duplicate items: %#v", indexRecord.Metadata)
	}
}
