package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverRuntimeVisibleSkills returns skills visible to the runtime after
// applying blocklist filtering and repo-before-user de-duplication.
func DiscoverRuntimeVisibleSkills(cfg Config) DiscoveryResult {
	roots := runtimeVisibleSkillRoots(cfg.ProjectRoot)
	blocklist := skillBlocklistSet(cfg.SkillBlocklist)
	return discoverRuntimeVisibleSkills(roots, blocklist)
}

// SearchRuntimeVisibleSkills returns runtime-visible skills filtered by query.
func SearchRuntimeVisibleSkills(cfg Config, query string) DiscoveryResult {
	result := DiscoverRuntimeVisibleSkills(cfg)
	if strings.TrimSpace(query) == "" {
		return result
	}
	filtered := make([]Skill, 0, len(result.Skills))
	for _, skill := range result.Skills {
		if skillMatchesQuery(skill, query) {
			filtered = append(filtered, skill)
		}
	}
	result.Skills = filtered
	return result
}

func discoverRuntimeVisibleSkills(roots skillRoots, blocklist map[string]bool) DiscoveryResult {
	result := DiscoveryResult{
		Skills: make([]Skill, 0, 8),
		Errors: make([]DiscoveryError, 0, 2),
	}
	seen := make(map[string]bool, 8)
	appendRuntimeVisibleSkills(&result, seen, blocklist, skillSourceRepo, roots.Repo)
	appendRuntimeVisibleSkills(&result, seen, blocklist, skillSourceUser, roots.User)
	sort.Slice(result.Skills, func(i, j int) bool {
		return result.Skills[i].Name < result.Skills[j].Name
	})
	sort.Slice(result.Errors, func(i, j int) bool {
		return result.Errors[i].Path < result.Errors[j].Path
	})
	return result
}

func appendRuntimeVisibleSkills(
	result *DiscoveryResult,
	seen map[string]bool,
	blocklist map[string]bool,
	source string,
	root string,
) {
	if result == nil || strings.TrimSpace(root) == "" {
		return
	}
	discovery := NewCatalogWithRoots([]string{root}).Discover()
	result.Errors = append(result.Errors, discovery.Errors...)
	for _, skill := range discovery.Skills {
		managed, err := managedSkillFromDiscovery(source, root, skill)
		if err != nil {
			result.Errors = append(result.Errors, DiscoveryError{
				Path:  skill.Path,
				Error: err.Error(),
			})
			continue
		}
		if blocklist[managed.ID] {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result.Skills = append(result.Skills, skill)
	}
}

func runtimeVisibleSkillRoots(projectRoot string) skillRoots {
	homeDir, _ := os.UserHomeDir()
	userRoot := ""
	if trimmedHome := strings.TrimSpace(homeDir); trimmedHome != "" {
		userRoot = filepath.Join(trimmedHome, ".ghost-os", "skills")
	}
	return skillRoots{
		Repo: filepath.Join(resolveRepoRoot(projectRoot), ".agents", "skills"),
		User: userRoot,
	}
}
