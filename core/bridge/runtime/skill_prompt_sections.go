package runtime

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/skills"
)

const (
	noVisibleSkillsAvailable = "- No visible skills available."
	loadSkillUsageHint       = `- Load a skill with sfind(action="load", skill_names=["skill_name"]).`
)

func formatVisibleSkillsCatalogFromDiscovery(
	discovery skills.DiscoveryResult,
) string {
	lines := []string{loadSkillUsageHint}
	lines = append(lines, visibleSkillLines(discovery.Skills)...)
	if len(discovery.Errors) > 0 {
		lines = append(lines, formatDiscoveryErrorBlock(discovery.Errors))
	}
	return strings.Join(lines, "\n")
}

func visibleSkillLines(items []skills.Skill) []string {
	if len(items) == 0 {
		return []string{noVisibleSkillsAvailable}
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Name, item.Description))
	}
	return lines
}

func skillRuntimeConfig(cfg Config) skills.Config {
	return skills.Config{
		ProjectRoot:    cfg.ProjectRoot,
		SkillBlocklist: append([]string(nil), cfg.SkillBlocklist...),
	}
}

func visibleSkillNames(cfg Config) []string {
	discovery := skills.DiscoverRuntimeVisibleSkills(skillRuntimeConfig(cfg))
	if len(discovery.Skills) == 0 {
		return nil
	}
	names := make([]string, 0, len(discovery.Skills))
	for _, item := range discovery.Skills {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
