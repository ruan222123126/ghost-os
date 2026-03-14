package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

const (
	memoryManageOperationRead   = "read"
	memoryManageOperationCreate = "create"
	memoryManageOperationUpdate = "update"
	memoryManageOperationDelete = "delete"
	memoryManageOperationSearch = "search"
	memoryManageOperationList   = "list"

	systemIndexURI  = "system://index"
	systemRecentURI = "system://recent"

	defaultMemoryLimit  = 50
	defaultMemoryOffset = 0
)

type MemoryManageTool struct {
	store *memorystore.Store
}

type memoryManageArgs struct {
	Operation string         `json:"operation"`
	URI       *string        `json:"uri,omitempty"`
	Content   *string        `json:"content,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Prefix    *string        `json:"prefix,omitempty"`
	Query     *string        `json:"query,omitempty"`
	Limit     *int           `json:"limit,omitempty"`
	Offset    *int           `json:"offset,omitempty"`
}

type memoryDeleteResult struct {
	URI     string `json:"uri"`
	Deleted bool   `json:"deleted"`
}

type memoryListResult struct {
	Items  []memorystore.Record `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

func NewMemoryManageTool(store *memorystore.Store) *MemoryManageTool {
	return &MemoryManageTool{store: store}
}

func NewMemoryManageToolFromEnv() (*MemoryManageTool, error) {
	store, err := memorystore.NewStoreFromEnv()
	if err != nil {
		return nil, err
	}
	return NewMemoryManageTool(store), nil
}

func (t *MemoryManageTool) Close() error {
	if t == nil || t.store == nil {
		return nil
	}
	return t.store.Close()
}

func (MemoryManageTool) Name() string {
	return "memory_manage"
}

func (MemoryManageTool) Description() string {
	return "Manage persistent memory entries by URI using create, read, update, delete, list, and search. Supports system://index and system://recent for read-only listings."
}

func (MemoryManageTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"operation":{"type":"string","enum":["read","create","update","delete","search","list"]},
			"uri":{"type":"string","description":"URI for read/create/update/delete. Supports system://index and system://recent."},
			"content":{"type":"string","description":"Content for create/update."},
			"metadata":{"type":"object","description":"Optional JSON metadata object."},
			"prefix":{"type":"string","description":"URI prefix for list."},
			"query":{"type":"string","description":"Search term for content LIKE query."},
			"limit":{"type":"integer","minimum":1,"description":"Max results for list/search/system index reads (default: 50)."},
			"offset":{"type":"integer","minimum":0,"description":"Offset for list/search/system index reads (default: 0)."}
		},
		"required":["operation"],
		"additionalProperties":false
	}`)
}

func (t *MemoryManageTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("memory store is not configured")
	}
	var args memoryManageArgs
	if err := decodeMemoryManageArgs(argsJSON, &args); err != nil {
		return "", err
	}
	operation := strings.ToLower(strings.TrimSpace(args.Operation))
	switch operation {
	case memoryManageOperationRead:
		return t.executeRead(ctx, args)
	case memoryManageOperationCreate:
		return t.executeCreate(ctx, args)
	case memoryManageOperationUpdate:
		return t.executeUpdate(ctx, args)
	case memoryManageOperationDelete:
		return t.executeDelete(ctx, args)
	case memoryManageOperationSearch:
		return t.executeSearch(ctx, args)
	case memoryManageOperationList:
		return t.executeList(ctx, args)
	default:
		return "", fmt.Errorf("unsupported operation %q", strings.TrimSpace(args.Operation))
	}
}

func (t *MemoryManageTool) executeRead(ctx context.Context, args memoryManageArgs) (string, error) {
	uri, err := requireMemoryURI(args.URI, "read")
	if err != nil {
		return "", err
	}
	limit, offset, err := resolveMemoryPaging(args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	switch uri {
	case systemIndexURI:
		return t.executeSystemIndex(ctx, limit, offset)
	case systemRecentURI:
		return t.executeSystemRecent(ctx, limit, offset)
	}
	record, err := t.store.Read(ctx, uri)
	if err != nil {
		return "", mapMemoryStoreError(err, uri)
	}
	return marshalMemoryManageOutput(record)
}

func (t *MemoryManageTool) executeCreate(ctx context.Context, args memoryManageArgs) (string, error) {
	uri, err := requireMemoryURI(args.URI, "create")
	if err != nil {
		return "", err
	}
	if err := ensureWritableMemoryURI(uri); err != nil {
		return "", err
	}
	content, err := requireMemoryContent(args.Content, "create")
	if err != nil {
		return "", err
	}
	record, err := t.store.Create(ctx, uri, content, args.Metadata)
	if err != nil {
		return "", mapMemoryStoreError(err, uri)
	}
	return marshalMemoryManageOutput(record)
}

func (t *MemoryManageTool) executeUpdate(ctx context.Context, args memoryManageArgs) (string, error) {
	uri, err := requireMemoryURI(args.URI, "update")
	if err != nil {
		return "", err
	}
	if err := ensureWritableMemoryURI(uri); err != nil {
		return "", err
	}
	content, err := requireMemoryContent(args.Content, "update")
	if err != nil {
		return "", err
	}
	record, err := t.store.Update(ctx, uri, content, args.Metadata)
	if err != nil {
		return "", mapMemoryStoreError(err, uri)
	}
	return marshalMemoryManageOutput(record)
}

func (t *MemoryManageTool) executeDelete(ctx context.Context, args memoryManageArgs) (string, error) {
	uri, err := requireMemoryURI(args.URI, "delete")
	if err != nil {
		return "", err
	}
	if err := ensureWritableMemoryURI(uri); err != nil {
		return "", err
	}
	deleted, err := t.store.Delete(ctx, uri)
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryDeleteResult{URI: uri, Deleted: deleted})
}

func (t *MemoryManageTool) executeSearch(ctx context.Context, args memoryManageArgs) (string, error) {
	query := strings.TrimSpace(optionalStringValue(args.Query))
	if query == "" {
		return "", fmt.Errorf("query is required for search")
	}
	limit, offset, err := resolveMemoryPaging(args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	items, total, err := t.store.Search(ctx, query, limit, offset)
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryListResult{Items: items, Total: total, Limit: limit, Offset: offset})
}

func (t *MemoryManageTool) executeList(ctx context.Context, args memoryManageArgs) (string, error) {
	if args.Prefix == nil {
		return "", fmt.Errorf("prefix is required for list")
	}
	limit, offset, err := resolveMemoryPaging(args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	items, total, err := t.store.List(ctx, strings.TrimSpace(*args.Prefix), limit, offset)
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryListResult{Items: items, Total: total, Limit: limit, Offset: offset})
}

func (t *MemoryManageTool) executeSystemIndex(ctx context.Context, limit int, offset int) (string, error) {
	items, total, err := t.store.ListAllURIs(ctx, limit, offset)
	if err != nil {
		return "", err
	}
	record := buildSystemRecord(systemIndexURI, items, total, limit, offset)
	return marshalMemoryManageOutput(record)
}

func (t *MemoryManageTool) executeSystemRecent(ctx context.Context, limit int, offset int) (string, error) {
	items, total, err := t.store.ListRecentURIs(ctx, limit, offset)
	if err != nil {
		return "", err
	}
	record := buildSystemRecord(systemRecentURI, items, total, limit, offset)
	return marshalMemoryManageOutput(record)
}
