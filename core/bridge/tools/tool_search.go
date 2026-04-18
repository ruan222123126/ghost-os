package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/skills"
	"ghost-os/bridge/tools/internal/tooljson"
)

const (
	toolSearchActionSearch = "search"
	toolSearchActionLoad   = "load"
	toolSearchActionUnload = "unload"
	toolSearchActionList   = "list"
	toolSearchKindTool     = "tool"
	toolSearchKindSkill    = "skill"
)

type ToolSearchTool struct {
	catalog      ToolCatalog
	visibility   VisibilityOptions
	idleTurns    int
	skillCatalog *skills.Catalog
}

type ToolSearchOptions struct {
	ProjectRoot  string
	SkillCatalog *skills.Catalog
}

type toolSearchArgs struct {
	Action    string   `json:"action"`
	Kind      string   `json:"kind,omitempty"`
	Query     string   `json:"query,omitempty"`
	ToolNames []string `json:"tool_names,omitempty"`
}

type toolSearchItem struct {
	Kind               string                     `json:"kind"`
	Name               string                     `json:"name"`
	Summary            string                     `json:"summary"`
	Path               string                     `json:"path,omitempty"`
	Error              string                     `json:"error,omitempty"`
	Status             string                     `json:"status,omitempty"`
	AvailableNow       bool                       `json:"available_now,omitempty"`
	AvailableNextTurn  bool                       `json:"available_next_turn,omitempty"`
	RemainingIdleTurns int                        `json:"remaining_idle_turns,omitempty"`
	Dependencies       []toolSearchDependencyItem `json:"dependencies,omitempty"`
}

type toolSearchPayload struct {
	Action string            `json:"action"`
	Kind   string            `json:"kind"`
	Items  []toolSearchItem  `json:"items,omitempty"`
	Errors []toolSearchError `json:"errors,omitempty"`
}

type toolSearchDependencyItem struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type toolSearchError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func NewToolSearchTool(
	catalog ToolCatalog,
	visibility VisibilityOptions,
	idleTurns int,
	options ...ToolSearchOptions,
) Tool {
	resolved := resolveToolSearchOptions(options)
	return &ToolSearchTool{
		catalog:      catalog,
		visibility:   visibility,
		idleTurns:    idleTurns,
		skillCatalog: resolveSkillCatalog(resolved),
	}
}

func (ToolSearchTool) Name() string {
	return ToolSearchToolName
}

func (ToolSearchTool) Description() string {
	return "Find optional tools or skills, and load or unload them for this session."
}

func (ToolSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["search","load","unload","list"]},
			"kind":{"type":"string","enum":["tool","skill"],"description":"Optional target kind; default is tool."},
			"query":{"type":"string","description":"Optional search text for action=search."},
			"tool_names":{
				"type":"array",
				"description":"Names to load or unload.",
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

	switch normalizeToolSearchKind(args.Kind) {
	case toolSearchKindTool:
		return t.executeToolAction(args, sess)
	case toolSearchKindSkill:
		return t.executeSkillAction(args, sess)
	default:
		return "", fmt.Errorf("kind must be one of: tool, skill")
	}
}

func (t *ToolSearchTool) executeToolAction(args toolSearchArgs, sess *session.Session) (string, error) {
	switch strings.ToLower(strings.TrimSpace(args.Action)) {
	case toolSearchActionSearch:
		return t.searchTools(args.Query, sess)
	case toolSearchActionLoad:
		return t.loadTools(sess, args.ToolNames)
	case toolSearchActionUnload:
		return t.unloadTools(sess, args.ToolNames)
	case toolSearchActionList:
		return t.listTools(sess)
	default:
		return "", fmt.Errorf("action must be one of: search, load, unload, list")
	}
}

func (t *ToolSearchTool) searchTools(query string, sess *session.Session) (string, error) {
	items := make([]toolSearchItem, 0)
	for _, name := range SearchCandidateToolNames(t.availableToolNames(), t.loadedToolNames(sess), t.visibility) {
		metadata, ok := ToolMetadataByName(name)
		if !ok || !toolMatchesQuery(metadata, query) {
			continue
		}
		items = append(items, toolSearchItem{
			Kind:    toolSearchKindTool,
			Name:    name,
			Summary: toolShortDescription(name, metadata.ShortDesc),
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionSearch, Kind: toolSearchKindTool, Items: items})
}

func (t *ToolSearchTool) loadTools(sess *session.Session, toolNames []string) (string, error) {
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
			Kind:              toolSearchKindTool,
			Name:              name,
			Summary:           toolShortDescription(name, ""),
			Status:            loadStatus(result.AlreadyLoaded),
			AvailableNow:      availableNow,
			AvailableNextTurn: availableNextTurn(result.Load, sess.TurnIndex, t.idleTurns),
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionLoad, Kind: toolSearchKindTool, Items: items})
}

func (t *ToolSearchTool) unloadTools(sess *session.Session, toolNames []string) (string, error) {
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
			Kind:    toolSearchKindTool,
			Name:    name,
			Summary: toolShortDescription(name, ""),
			Status:  "unloaded",
		})
	}
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionUnload, Kind: toolSearchKindTool, Items: items})
}

func (t *ToolSearchTool) listTools(sess *session.Session) (string, error) {
	loads := sess.DynamicToolLoadsSnapshot()
	if len(loads) == 0 {
		return tooljson.Encode(toolSearchPayload{Action: toolSearchActionList, Kind: toolSearchKindTool})
	}

	items := make([]toolSearchItem, 0, len(loads))
	for _, load := range loads {
		items = append(items, toolSearchItem{
			Kind:               toolSearchKindTool,
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
	return tooljson.Encode(toolSearchPayload{Action: toolSearchActionList, Kind: toolSearchKindTool, Items: items})
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

func resolveToolSearchOptions(options []ToolSearchOptions) ToolSearchOptions {
	if len(options) == 0 {
		return ToolSearchOptions{}
	}
	return options[0]
}

func resolveSkillCatalog(options ToolSearchOptions) *skills.Catalog {
	if options.SkillCatalog != nil {
		return options.SkillCatalog
	}
	return skills.NewCatalog(options.ProjectRoot)
}

func normalizeToolSearchKind(raw string) string {
	kind := strings.ToLower(strings.TrimSpace(raw))
	if kind == "" {
		return toolSearchKindTool
	}
	return kind
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
