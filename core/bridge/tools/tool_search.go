package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/tooljson"
)

const (
	toolSearchActionSearch = "search"
	toolSearchActionLoad   = "load"
	toolSearchActionUnload = "unload"
	toolSearchActionList   = "list"
	minToolSearchTermLen   = 2
)

var toolSearchWordSplitPattern = regexp.MustCompile(`[^a-z0-9]+`)

var toolSearchStopWords = map[string]bool{
	"a":            true,
	"an":           true,
	"and":          true,
	"availability": true,
	"available":    true,
	"can":          true,
	"check":        true,
	"current":      true,
	"currently":    true,
	"find":         true,
	"for":          true,
	"in":           true,
	"is":           true,
	"look":         true,
	"of":           true,
	"only":         true,
	"or":           true,
	"the":          true,
	"tool":         true,
	"tools":        true,
	"use":          true,
	"using":        true,
	"visible":      true,
}

type ToolSearchTool struct {
	catalog    ToolCatalog
	visibility VisibilityOptions
	idleTurns  int
}

type toolSearchArgs struct {
	Action    string   `json:"action"`
	Query     string   `json:"query,omitempty"`
	ToolNames []string `json:"tool_names,omitempty"`
}

type toolSearchItem struct {
	Name               string `json:"name"`
	Summary            string `json:"summary"`
	Status             string `json:"status,omitempty"`
	AvailableNow       bool   `json:"available_now,omitempty"`
	AvailableNextTurn  bool   `json:"available_next_turn,omitempty"`
	RemainingIdleTurns int    `json:"remaining_idle_turns,omitempty"`
}

type toolSearchPayload struct {
	Action string           `json:"action"`
	Items  []toolSearchItem `json:"items,omitempty"`
}

func NewToolSearchTool(catalog ToolCatalog, visibility VisibilityOptions, idleTurns int) Tool {
	return &ToolSearchTool{
		catalog:    catalog,
		visibility: visibility,
		idleTurns:  idleTurns,
	}
}

func (ToolSearchTool) Name() string {
	return ToolSearchToolName
}

func (ToolSearchTool) Description() string {
	return "Find optional tools and load or unload them for this session."
}

func (ToolSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["search","load","unload","list"]},
			"query":{"type":"string","description":"Optional search text for action=search."},
			"tool_names":{
				"type":"array",
				"description":"Tool names to load or unload.",
				"items":{"type":"string"}
			}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t *ToolSearchTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("%s requires an active session", ToolSearchToolName)
	}

	var args toolSearchArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	action := strings.ToLower(strings.TrimSpace(args.Action))
	switch action {
	case toolSearchActionSearch:
		return t.search(args.Query, sess)
	case toolSearchActionLoad:
		return t.load(sess, args.ToolNames)
	case toolSearchActionUnload:
		return t.unload(sess, args.ToolNames)
	case toolSearchActionList:
		return t.list(sess)
	default:
		return "", fmt.Errorf("action must be one of: search, load, unload, list")
	}
}

func (t *ToolSearchTool) search(query string, sess *session.Session) (string, error) {
	items := make([]toolSearchItem, 0)
	for _, name := range SearchCandidateToolNames(t.availableToolNames(), t.loadedToolNames(sess), t.visibility) {
		metadata, ok := ToolMetadataByName(name)
		if !ok || !toolMatchesQuery(metadata, query) {
			continue
		}
		items = append(items, toolSearchItem{
			Name:    name,
			Summary: toolShortDescription(name, metadata.ShortDesc),
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionSearch, Items: items})
}

func (t *ToolSearchTool) load(sess *session.Session, toolNames []string) (string, error) {
	names := normalizeVisibleToolNames(toolNames)
	if len(names) == 0 {
		return "", fmt.Errorf("tool_names is required for load")
	}

	candidateSet := toolNameSet(SearchCandidateToolNames(t.availableToolNames(), t.loadedToolNames(sess), t.visibility))
	loadedSet := toolNameSet(t.loadedToolNames(sess))
	items := make([]toolSearchItem, 0, len(names))
	for _, name := range names {
		if !candidateSet[name] && !loadedSet[name] {
			return "", fmt.Errorf("tool %q is not loadable in this session", name)
		}
		result := sess.EnsureDynamicToolLoaded(name, ToolSearchToolName)
		availableNow := result.Load.VisibleForTurn(sess.TurnIndex) && !result.Load.ExpiredAtTurn(sess.TurnIndex, t.idleTurns)
		items = append(items, toolSearchItem{
			Name:              name,
			Summary:           toolShortDescription(name, ""),
			Status:            loadStatus(result.AlreadyLoaded),
			AvailableNow:      availableNow,
			AvailableNextTurn: availableNextTurn(result.Load, sess.TurnIndex, t.idleTurns),
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionLoad, Items: items})
}

func (t *ToolSearchTool) unload(sess *session.Session, toolNames []string) (string, error) {
	names := normalizeVisibleToolNames(toolNames)
	if len(names) == 0 {
		return "", fmt.Errorf("tool_names is required for unload")
	}

	items := make([]toolSearchItem, 0, len(names))
	for _, name := range names {
		if !sess.UnloadDynamicTool(name) {
			return "", fmt.Errorf("tool %q is not currently loaded", name)
		}
		items = append(items, toolSearchItem{
			Name:    name,
			Summary: toolShortDescription(name, ""),
			Status:  "unloaded",
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionUnload, Items: items})
}

func (t *ToolSearchTool) list(sess *session.Session) (string, error) {
	loads := sess.DynamicToolLoadsSnapshot()
	if len(loads) == 0 {
		return tooljson.Encode(toolSearchPayload{Action: toolSearchActionList})
	}

	items := make([]toolSearchItem, 0, len(loads))
	for _, load := range loads {
		items = append(items, toolSearchItem{
			Name:               load.ToolName,
			Summary:            toolShortDescription(load.ToolName, ""),
			Status:             listStatus(load, sess.TurnIndex, t.idleTurns),
			AvailableNow:       load.VisibleForTurn(sess.TurnIndex) && !load.ExpiredAtTurn(sess.TurnIndex, t.idleTurns),
			AvailableNextTurn:  availableNextTurn(load, sess.TurnIndex, t.idleTurns),
			RemainingIdleTurns: load.RemainingIdleTurns(sess.TurnIndex, t.idleTurns),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionList, Items: items})
}

func (t *ToolSearchTool) availableToolNames() []string {
	return CatalogToolNames(t.catalog)
}

func (t *ToolSearchTool) loadedToolNames(sess *session.Session) []string {
	loads := sess.DynamicToolLoadsSnapshot()
	names := make([]string, 0, len(loads))
	for _, load := range loads {
		if load.ToolName != "" {
			names = append(names, load.ToolName)
		}
	}
	sort.Strings(names)
	return names
}

func loadStatus(alreadyLoaded bool) string {
	if alreadyLoaded {
		return "already_loaded"
	}
	return "loaded"
}

func listStatus(load session.DynamicToolLoad, currentTurn int, idleTurns int) string {
	switch {
	case load.ExpiredAtTurn(currentTurn, idleTurns):
		return "expired"
	case load.VisibleForTurn(currentTurn):
		return "active"
	case load.LoadedAtTurn > currentTurn:
		return "pending"
	default:
		return "active"
	}
}

func availableNextTurn(load session.DynamicToolLoad, currentTurn int, idleTurns int) bool {
	return !load.VisibleForTurn(currentTurn) &&
		!load.ExpiredAtTurn(currentTurn, idleTurns) &&
		load.LoadedAtTurn > currentTurn
}

func toolMatchesQuery(metadata ToolMetadata, query string) bool {
	normalizedQuery := normalizeToolSearchText(query)
	if normalizedQuery == "" {
		return true
	}
	document := toolSearchDocument(metadata)
	if strings.Contains(document, normalizedQuery) {
		return true
	}
	terms := toolSearchQueryTerms(normalizedQuery)
	if len(terms) == 0 {
		return false
	}
	if toolSearchPrimaryFieldMatch(metadata, terms) {
		return true
	}
	return toolSearchDocumentMatchCount(document, terms) >= 2
}

func toolSearchDocument(metadata ToolMetadata) string {
	parts := make([]string, 0, len(metadata.Tags)+2)
	parts = append(parts, normalizeToolSearchText(metadata.Name))
	parts = append(parts, normalizeToolSearchText(metadata.ShortDesc))
	for _, tag := range metadata.Tags {
		parts = append(parts, normalizeToolSearchText(tag))
	}
	return strings.Join(parts, " ")
}

func toolSearchPrimaryFieldMatch(metadata ToolMetadata, terms []string) bool {
	name := normalizeToolSearchText(metadata.Name)
	tags := normalizeToolSearchText(strings.Join(metadata.Tags, " "))
	for _, term := range terms {
		if strings.Contains(name, term) || strings.Contains(tags, term) {
			return true
		}
	}
	return false
}

func toolSearchQueryTerms(query string) []string {
	words := strings.Fields(query)
	terms := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	for _, word := range words {
		if len(word) < minToolSearchTermLen || toolSearchStopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		terms = append(terms, word)
	}
	return terms
}

func toolSearchDocumentMatchCount(document string, terms []string) int {
	count := 0
	for _, term := range terms {
		if strings.Contains(document, term) {
			count++
		}
	}
	return count
}

func normalizeToolSearchText(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}
	normalized := toolSearchWordSplitPattern.ReplaceAllString(trimmed, " ")
	return strings.Join(strings.Fields(normalized), " ")
}
