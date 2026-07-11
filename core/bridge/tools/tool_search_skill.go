package tools

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/skills"
	"ghost-os/bridge/tools/internal/tooljson"
)

func (t *ToolSearchTool) executeSkillAction(args toolSearchArgs, sess *session.Session) (string, error) {
	switch strings.ToLower(strings.TrimSpace(args.Action)) {
	case toolSearchActionSearch:
		return t.searchSkills(args.Query, sess)
	case toolSearchActionLoad:
		return t.loadSkills(sess, args.SkillNames)
	case toolSearchActionUnload:
		return t.unloadSkills(sess, args.SkillNames)
	case toolSearchActionList:
		return t.listSkills(sess)
	default:
		return "", fmt.Errorf("action must be one of: search, load, unload, list")
	}
}

func (t *ToolSearchTool) searchSkills(query string, sess *session.Session) (string, error) {
	discovery := t.searchableSkillCatalog(query)
	loaded := toolNameSet(t.loadedSkillNames(sess))
	items := make([]toolSearchItem, 0, len(discovery.Skills))
	for _, skill := range discovery.Skills {
		if loaded[skill.Name] {
			continue
		}
		items = append(items, toolSearchItem{
			Kind:    toolSearchKindSkill,
			Name:    skill.Name,
			Summary: strings.TrimSpace(skill.Description),
			Path:    skill.Path,
		})
	}
	return encodeSkillPayload(toolSearchActionSearch, items, discovery.Errors)
}

func (t *ToolSearchTool) loadSkills(sess *session.Session, skillNames []string) (string, error) {
	names := normalizeVisibleToolNames(skillNames)
	if len(names) == 0 {
		return "", fmt.Errorf("skill_names is required for load")
	}
	discovery := t.discoverSkills()
	skillMap := indexSkillsByName(discovery.Skills)
	loadedSet := toolNameSet(t.loadedSkillNames(sess))
	items := make([]toolSearchItem, 0, len(names))
	for _, rawName := range names {
		item, err := t.loadOneSkill(sess, rawName, skillMap, loadedSet)
		if err != nil {
			return "", err
		}
		items = append(items, item)
	}
	return encodeSkillPayload(toolSearchActionLoad, items, discovery.Errors)
}

func (t *ToolSearchTool) loadOneSkill(
	sess *session.Session,
	rawName string,
	skillMap map[string]skills.Skill,
	loadedSet map[string]bool,
) (toolSearchItem, error) {
	name := strings.TrimSpace(rawName)
	lookupKey := strings.ToLower(name)
	skill, exists := skillMap[lookupKey]
	if !exists && !loadedSet[name] {
		return toolSearchItem{}, fmt.Errorf("skill %q is not loadable in this session", name)
	}
	if exists {
		name = skill.Name
	}
	result := sess.EnsureDynamicSkillLoaded(name, ToolSearchToolName)
	return toolSearchItem{
		Kind:              toolSearchKindSkill,
		Name:              name,
		Summary:           skillSummary(skill, exists),
		Path:              skillPath(skill, exists),
		Status:            loadStatus(result.AlreadyLoaded),
		AvailableNow:      result.Load.VisibleForTurn(sess.TurnIndex) && !result.Load.ExpiredAtTurn(sess.TurnIndex, t.idleTurns),
		AvailableNextTurn: availableNextTurnForSkill(result.Load, sess.TurnIndex, t.idleTurns),
	}, nil
}

func (t *ToolSearchTool) unloadSkills(sess *session.Session, skillNames []string) (string, error) {
	names := normalizeVisibleToolNames(skillNames)
	if len(names) == 0 {
		return "", fmt.Errorf("skill_names is required for unload")
	}
	items := make([]toolSearchItem, 0, len(names))
	for _, name := range names {
		if !sess.UnloadDynamicSkill(name) {
			return "", fmt.Errorf("skill %q is not currently loaded", name)
		}
		items = append(items, toolSearchItem{
			Kind:    toolSearchKindSkill,
			Name:    name,
			Summary: "Loaded skill.",
			Status:  "unloaded",
		})
	}
	return encodeSkillPayload(toolSearchActionUnload, items, nil)
}

func (t *ToolSearchTool) listSkills(sess *session.Session) (string, error) {
	discovery := t.discoverSkills()
	loads := sess.DynamicSkillLoadsSnapshot()
	if len(loads) == 0 {
		return encodeSkillPayload(toolSearchActionList, nil, discovery.Errors)
	}
	skillMap := indexSkillsByName(discovery.Skills)
	items := make([]toolSearchItem, 0, len(loads))
	for _, load := range loads {
		items = append(items, skillListItem(load, sess.TurnIndex, t.idleTurns, skillMap))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return encodeSkillPayload(toolSearchActionList, items, discovery.Errors)
}

func skillListItem(
	load session.DynamicSkillLoad,
	currentTurn int,
	idleTurns int,
	skillMap map[string]skills.Skill,
) toolSearchItem {
	skill, exists := skillMap[strings.ToLower(strings.TrimSpace(load.SkillName))]
	return toolSearchItem{
		Kind:               toolSearchKindSkill,
		Name:               load.SkillName,
		Summary:            skillSummary(skill, exists),
		Path:               skillPath(skill, exists),
		Status:             listSkillStatus(load, currentTurn, idleTurns),
		AvailableNow:       load.VisibleForTurn(currentTurn) && !load.ExpiredAtTurn(currentTurn, idleTurns),
		AvailableNextTurn:  availableNextTurnForSkill(load, currentTurn, idleTurns),
		RemainingIdleTurns: load.RemainingIdleTurns(currentTurn, idleTurns),
	}
}

func loadedSkillNames(sess *session.Session) []string {
	if sess == nil {
		return nil
	}
	loads := sess.DynamicSkillLoadsSnapshot()
	names := make([]string, 0, len(loads))
	for _, load := range loads {
		name := strings.TrimSpace(load.SkillName)
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (t *ToolSearchTool) loadedSkillNames(sess *session.Session) []string {
	return loadedSkillNames(sess)
}

func indexSkillsByName(items []skills.Skill) map[string]skills.Skill {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]skills.Skill, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Name))
		if key != "" {
			out[key] = item
		}
	}
	return out
}

func encodeSkillPayload(action string, items []toolSearchItem, errors []skills.DiscoveryError) (string, error) {
	payload := toolSearchPayload{
		Action: action,
		Kind:   toolSearchKindSkill,
		Items:  items,
		Errors: convertSkillDiscoveryErrors(errors),
	}
	return tooljson.Encode(payload)
}

func convertSkillDiscoveryErrors(errors []skills.DiscoveryError) []toolSearchError {
	if len(errors) == 0 {
		return nil
	}
	out := make([]toolSearchError, 0, len(errors))
	for _, item := range errors {
		out = append(out, toolSearchError{
			Path:  item.Path,
			Error: item.Error,
		})
	}
	return out
}

func skillSummary(skill skills.Skill, exists bool) string {
	if !exists {
		return "Loaded skill."
	}
	return strings.TrimSpace(skill.Description)
}

func skillPath(skill skills.Skill, exists bool) string {
	if !exists {
		return ""
	}
	return skill.Path
}

func listSkillStatus(load session.DynamicSkillLoad, currentTurn int, idleTurns int) string {
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

func availableNextTurnForSkill(load session.DynamicSkillLoad, currentTurn int, idleTurns int) bool {
	return !load.VisibleForTurn(currentTurn) &&
		!load.ExpiredAtTurn(currentTurn, idleTurns) &&
		load.LoadedAtTurn > currentTurn
}

func (t *ToolSearchTool) discoverSkills() skills.DiscoveryResult {
	if t != nil && t.skillCatalog != nil {
		return t.skillCatalog.Discover()
	}
	return skills.DiscoverRuntimeVisibleSkills(t.skillConfig)
}

func (t *ToolSearchTool) searchableSkillCatalog(query string) skills.DiscoveryResult {
	if t != nil && t.skillCatalog != nil {
		return t.skillCatalog.Search(query)
	}
	return skills.SearchRuntimeVisibleSkills(t.skillConfig, query)
}
