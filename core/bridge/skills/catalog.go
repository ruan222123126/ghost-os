package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var discoveryCache sync.Map

type Catalog struct {
	roots []string
}

func NewCatalog(projectRoot string) *Catalog {
	homeDir, _ := os.UserHomeDir()
	return NewCatalogWithRoots(defaultRoots(projectRoot, homeDir))
}

func NewCatalogWithRoots(roots []string) *Catalog {
	return &Catalog{roots: normalizeRoots(roots)}
}

func (c *Catalog) Discover() DiscoveryResult {
	if c == nil {
		return DiscoveryResult{}
	}
	key := cacheKey(c.roots)
	if cached, ok := discoveryCache.Load(key); ok {
		result, _ := cached.(DiscoveryResult)
		return cloneDiscoveryResult(result)
	}
	return c.ForceReload()
}

func (c *Catalog) ForceReload() DiscoveryResult {
	if c == nil {
		return DiscoveryResult{}
	}
	result := discoverSkills(c.roots)
	discoveryCache.Store(cacheKey(c.roots), result)
	return cloneDiscoveryResult(result)
}

func (c *Catalog) Search(query string) DiscoveryResult {
	result := c.Discover()
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

func (c *Catalog) SkillByName(name string) (Skill, bool) {
	target := strings.TrimSpace(name)
	if target == "" {
		return Skill{}, false
	}
	for _, skill := range c.Discover().Skills {
		if strings.EqualFold(skill.Name, target) {
			return skill, true
		}
	}
	return Skill{}, false
}

func defaultRoots(projectRoot string, homeDir string) []string {
	repoRoot := resolveRepoRoot(projectRoot)
	roots := []string{filepath.Join(repoRoot, ".agents", "skills")}
	if trimmedHome := strings.TrimSpace(homeDir); trimmedHome != "" {
		roots = append(roots, filepath.Join(trimmedHome, ".ghost-os", "skills"))
	}
	return normalizeRoots(roots)
}

func resolveRepoRoot(projectRoot string) string {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return ""
		}
		root = cwd
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return filepath.Clean(absolute)
}

func normalizeRoots(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(raw))
	roots := make([]string, 0, len(raw))
	for _, root := range raw {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		roots = append(roots, cleaned)
	}
	return roots
}

func cacheKey(roots []string) string {
	return strings.Join(normalizeRoots(roots), "\n")
}

func cloneDiscoveryResult(raw DiscoveryResult) DiscoveryResult {
	out := DiscoveryResult{}
	if len(raw.Skills) > 0 {
		out.Skills = make([]Skill, 0, len(raw.Skills))
		for _, skill := range raw.Skills {
			out.Skills = append(out.Skills, Skill{
				Name:        skill.Name,
				Description: skill.Description,
				Body:        skill.Body,
				Path:        skill.Path,
				Policy:      skill.Policy,
				Dependencies: Dependencies{
					Tools: append([]string(nil), skill.Dependencies.Tools...),
				},
			})
		}
	}
	if len(raw.Errors) > 0 {
		out.Errors = append([]DiscoveryError(nil), raw.Errors...)
	}
	return out
}

func resetDiscoveryCacheForTests() {
	discoveryCache = sync.Map{}
}
