package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/skills"
)

const noDynamicSkillsLoaded = "- No dynamic skills loaded."

func formatDynamicSkillContext(cfg Config, sess *session.Session, idleTurns int) string {
	return formatDynamicSkillContextFromDiscovery(
		sess,
		idleTurns,
		skills.DiscoverRuntimeVisibleSkills(skillRuntimeConfig(cfg)),
	)
}

func formatDynamicSkillContextFromDiscovery(
	sess *session.Session,
	idleTurns int,
	discovery skills.DiscoveryResult,
) string {
	if sess == nil {
		return noDynamicSkillsLoaded
	}
	loads := sess.DynamicSkillLoadsSnapshot()
	if len(loads) == 0 {
		return noDynamicSkillsLoaded
	}
	skillMap := indexSkillsByName(discovery.Skills)
	blocks := buildDynamicSkillBlocks(loads, sess.TurnIndex, idleTurns, skillMap)
	if len(discovery.Errors) > 0 {
		blocks = append(blocks, formatDiscoveryErrorBlock(discovery.Errors))
	}
	if len(blocks) == 0 {
		return noDynamicSkillsLoaded
	}
	return strings.Join(blocks, "\n\n")
}

func buildDynamicSkillBlocks(
	loads []session.DynamicSkillLoad,
	currentTurn int,
	idleTurns int,
	skillMap map[string]skills.Skill,
) []string {
	blocks := make([]string, 0, len(loads))
	for _, load := range loads {
		block := dynamicSkillContextBlock(load, currentTurn, idleTurns, skillMap)
		if strings.TrimSpace(block) != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func dynamicSkillContextBlock(
	load session.DynamicSkillLoad,
	currentTurn int,
	idleTurns int,
	skillMap map[string]skills.Skill,
) string {
	name := strings.TrimSpace(load.SkillName)
	if name == "" {
		return ""
	}
	switch {
	case load.ExpiredAtTurn(currentTurn, idleTurns):
		return fmt.Sprintf("- `%s` is expired and must be loaded again with `sfind(action=\"load\")`.", name)
	case !load.VisibleForTurn(currentTurn):
		return fmt.Sprintf("- `%s` is pending for a future turn.", name)
	}
	skill, ok := skillMap[strings.ToLower(name)]
	if !ok {
		return fmt.Sprintf("- `%s` is active, but its SKILL.md source is unavailable.", name)
	}
	body := strings.TrimSpace(skill.Body)
	if body == "" {
		return fmt.Sprintf("- `%s` is active, but its SKILL.md body is empty.", name)
	}
	return fmt.Sprintf("### Skill `%s`\n- Path: `%s`\n\n%s", skill.Name, skill.Path, body)
}

func indexSkillsByName(items []skills.Skill) map[string]skills.Skill {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]skills.Skill, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Name))
		if key == "" {
			continue
		}
		out[key] = item
	}
	return out
}

func formatDiscoveryErrorBlock(errors []skills.DiscoveryError) string {
	lines := make([]string, 0, len(errors)+1)
	lines = append(lines, "Skill discovery errors:")
	for _, item := range errors {
		lines = append(lines, fmt.Sprintf("- path=%s error=%s", item.Path, item.Error))
	}
	return strings.Join(lines, "\n")
}
