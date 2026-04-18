package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	skillFileName       = "SKILL.md"
	openAIMetadataPath  = "agents/openai.yaml"
	minSearchTermLength = 2
)

var searchWordSplitPattern = regexp.MustCompile(`[^a-z0-9]+`)
var dependencyToolNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type openAIConfig struct {
	Policy       openAIPolicy       `yaml:"policy"`
	Dependencies openAIDependencies `yaml:"dependencies"`
}

type openAIPolicy struct {
	AllowImplicitInvocation *bool `yaml:"allow_implicit_invocation"`
}

type openAIDependencies struct {
	Tools []string `yaml:"tools"`
}

func discoverSkills(roots []string) DiscoveryResult {
	result := DiscoveryResult{
		Skills: make([]Skill, 0, 8),
		Errors: make([]DiscoveryError, 0, 2),
	}
	seen := make(map[string]bool, 8)
	for _, root := range normalizeRoots(roots) {
		discoverSkillsFromRoot(root, seen, &result)
	}
	sort.Slice(result.Skills, func(i, j int) bool {
		return result.Skills[i].Name < result.Skills[j].Name
	})
	sort.Slice(result.Errors, func(i, j int) bool {
		return result.Errors[i].Path < result.Errors[j].Path
	})
	return result
}

func discoverSkillsFromRoot(root string, seen map[string]bool, result *DiscoveryResult) {
	if result == nil {
		return
	}
	if strings.TrimSpace(root) == "" {
		return
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return
		}
		result.Errors = append(result.Errors, DiscoveryError{Path: root, Error: err.Error()})
		return
	}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Errors = append(result.Errors, DiscoveryError{Path: path, Error: walkErr.Error()})
			return nil
		}
		if entry.IsDir() || entry.Name() != skillFileName {
			return nil
		}
		skill, err := parseSkillFile(path)
		if err != nil {
			result.Errors = append(result.Errors, DiscoveryError{Path: path, Error: err.Error()})
			return nil
		}
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		if key == "" || seen[key] {
			return nil
		}
		seen[key] = true
		result.Skills = append(result.Skills, skill)
		return nil
	})
}

func parseSkillFile(path string) (Skill, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Skill{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	frontmatterText, body, err := splitFrontmatterAndBody(string(raw))
	if err != nil {
		return Skill{}, err
	}
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatterText), &frontmatter); err != nil {
		return Skill{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	name := strings.TrimSpace(frontmatter.Name)
	description := strings.TrimSpace(frontmatter.Description)
	if name == "" {
		return Skill{}, fmt.Errorf("frontmatter field `name` is required")
	}
	if description == "" {
		return Skill{}, fmt.Errorf("frontmatter field `description` is required")
	}
	policy, deps, err := parseOptionalOpenAIConfig(filepath.Dir(path))
	if err != nil {
		return Skill{}, err
	}
	return Skill{
		Name:         name,
		Description:  description,
		Body:         body,
		Path:         filepath.Clean(path),
		Policy:       policy,
		Dependencies: deps,
	}, nil
}

func splitFrontmatterAndBody(raw string) (string, string, error) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("SKILL.md must start with YAML frontmatter delimiter `---`")
	}
	endIndex := findFrontmatterEnd(lines)
	if endIndex < 0 {
		return "", "", fmt.Errorf("SKILL.md frontmatter is missing closing `---` delimiter")
	}
	body := strings.TrimSpace(strings.Join(lines[endIndex+1:], "\n"))
	if body == "" {
		return "", "", fmt.Errorf("SKILL.md body is required")
	}
	return strings.Join(lines[1:endIndex], "\n"), body, nil
}

func findFrontmatterEnd(lines []string) int {
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return index
		}
	}
	return -1
}

func parseOptionalOpenAIConfig(skillDir string) (Policy, Dependencies, error) {
	path := filepath.Join(filepath.Clean(skillDir), openAIMetadataPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Policy{}, Dependencies{}, nil
		}
		return Policy{}, Dependencies{}, fmt.Errorf("read %s: %w", openAIMetadataPath, err)
	}
	var cfg openAIConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Policy{}, Dependencies{}, fmt.Errorf("parse %s: %w", openAIMetadataPath, err)
	}
	deps, err := normalizeDependencyTools(cfg.Dependencies.Tools)
	if err != nil {
		return Policy{}, Dependencies{}, err
	}
	policy := Policy{}
	if cfg.Policy.AllowImplicitInvocation != nil {
		policy.AllowImplicitInvocation = *cfg.Policy.AllowImplicitInvocation
	}
	return policy, deps, nil
}

func normalizeDependencyTools(raw []string) (Dependencies, error) {
	if len(raw) == 0 {
		return Dependencies{}, nil
	}
	seen := make(map[string]bool, len(raw))
	tools := make([]string, 0, len(raw))
	for index, item := range raw {
		name := strings.TrimSpace(item)
		if name == "" {
			return Dependencies{}, fmt.Errorf("dependencies.tools[%d] must not be empty", index)
		}
		if !dependencyToolNamePattern.MatchString(name) {
			return Dependencies{}, fmt.Errorf("dependencies.tools[%d] has invalid tool name %q", index, name)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		tools = append(tools, name)
	}
	return Dependencies{Tools: tools}, nil
}

func skillMatchesQuery(skill Skill, query string) bool {
	normalizedQuery := normalizeSearchText(query)
	if normalizedQuery == "" {
		return true
	}
	document := normalizeSearchText(skill.Name + " " + skill.Description + " " + skill.Body)
	if strings.Contains(document, normalizedQuery) {
		return true
	}
	return matchAnySearchTerm(document, normalizedQuery)
}

func matchAnySearchTerm(document string, normalizedQuery string) bool {
	for _, term := range strings.Fields(normalizedQuery) {
		if len(term) < minSearchTermLength {
			continue
		}
		if strings.Contains(document, term) {
			return true
		}
	}
	return false
}

func normalizeSearchText(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}
	normalized := searchWordSplitPattern.ReplaceAllString(trimmed, " ")
	return strings.Join(strings.Fields(normalized), " ")
}
