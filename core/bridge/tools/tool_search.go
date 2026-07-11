package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/skills"
)

const (
	toolSearchActionSearch = "search"
	toolSearchActionLoad   = "load"
	toolSearchActionUnload = "unload"
	toolSearchActionList   = "list"
	toolSearchKindSkill    = "skill"
)

type ToolSearchTool struct {
	idleTurns    int
	skillCatalog *skills.Catalog
	skillConfig  skills.Config
}

type ToolSearchOptions struct {
	ProjectRoot  string
	SkillCatalog *skills.Catalog
	SkillConfig  skills.Config
}

type toolSearchArgs struct {
	Action     string   `json:"action"`
	Kind       string   `json:"kind,omitempty"`
	Query      string   `json:"query,omitempty"`
	SkillNames []string `json:"skill_names,omitempty"`
}

type toolSearchItem struct {
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	Summary            string `json:"summary"`
	Path               string `json:"path,omitempty"`
	Error              string `json:"error,omitempty"`
	Status             string `json:"status,omitempty"`
	AvailableNow       bool   `json:"available_now,omitempty"`
	AvailableNextTurn  bool   `json:"available_next_turn,omitempty"`
	RemainingIdleTurns int    `json:"remaining_idle_turns,omitempty"`
}

type toolSearchPayload struct {
	Action string            `json:"action"`
	Kind   string            `json:"kind"`
	Items  []toolSearchItem  `json:"items,omitempty"`
	Errors []toolSearchError `json:"errors,omitempty"`
}

type toolSearchError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func NewToolSearchTool(
	_ ToolCatalog,
	_ VisibilityOptions,
	idleTurns int,
	options ...ToolSearchOptions,
) Tool {
	resolved := resolveToolSearchOptions(options)
	return &ToolSearchTool{
		idleTurns:    idleTurns,
		skillCatalog: resolveSkillCatalog(resolved),
		skillConfig:  resolveSkillConfig(resolved),
	}
}

func (ToolSearchTool) Name() string {
	return ToolSearchToolName
}

func (ToolSearchTool) Description() string {
	return "Manage dynamic skills from SKILL.md. 'search' finds them, 'load' applies them immediately for this session, 'unload' removes them, 'list' shows current state. Use ONLY when visible tools are insufficient."
}

func (ToolSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["search","load","unload","list"]},
			"query":{"type":"string"},
			"skill_names":{
				"type":"array",
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
	if kind := normalizeToolSearchKind(args.Kind); kind != "" && kind != toolSearchKindSkill {
		return "", fmt.Errorf("%s supports skills only", ToolSearchToolName)
	}
	return t.executeSkillAction(args, sess)
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
	return nil
}

func normalizeToolSearchKind(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func loadStatus(alreadyLoaded bool) string {
	if alreadyLoaded {
		return "already_loaded"
	}
	return "loaded"
}

func resolveSkillConfig(options ToolSearchOptions) skills.Config {
	if strings.TrimSpace(options.SkillConfig.ProjectRoot) != "" || len(options.SkillConfig.SkillBlocklist) > 0 {
		return options.SkillConfig
	}
	return skills.Config{ProjectRoot: options.ProjectRoot}
}
