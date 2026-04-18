package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogDiscoversSkillsAndDeduplicatesByName(t *testing.T) {
	resetDiscoveryCacheForTests()
	root := t.TempDir()
	repoRoot := filepath.Join(root, "repo")
	userRoot := filepath.Join(root, "home")
	repoSkills := filepath.Join(repoRoot, ".agents", "skills")
	userSkills := filepath.Join(userRoot, ".ghost-os", "skills")

	writeSkillFixture(t, filepath.Join(repoSkills, "deploy"), "deploy", "repo skill", "Run repo deployment.", "")
	writeSkillFixture(t, filepath.Join(userSkills, "deploy"), "deploy", "user skill", "Run user deployment.", "")

	catalog := NewCatalogWithRoots([]string{repoSkills, userSkills})
	result := catalog.ForceReload()
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected discovery errors: %+v", result.Errors)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected one deduped skill, got %+v", result.Skills)
	}
	skill := result.Skills[0]
	if skill.Name != "deploy" || skill.Description != "repo skill" {
		t.Fatalf("unexpected resolved skill: %+v", skill)
	}
	if !strings.Contains(skill.Path, filepath.Join(".agents", "skills")) {
		t.Fatalf("expected repo root skill to win, got %q", skill.Path)
	}
}

func TestCatalogParsesOptionalOpenAIConfig(t *testing.T) {
	resetDiscoveryCacheForTests()
	root := t.TempDir()
	repoSkills := filepath.Join(root, ".agents", "skills")
	openAIConfig := "policy:\n  allow_implicit_invocation: true\ndependencies:\n  tools:\n    - web_search\n    - script_exec\n"
	writeSkillFixture(t, filepath.Join(repoSkills, "research"), "research", "Research workflow", "Investigate with citations.", openAIConfig)

	catalog := NewCatalogWithRoots([]string{repoSkills})
	result := catalog.ForceReload()
	if len(result.Errors) != 0 || len(result.Skills) != 1 {
		t.Fatalf("unexpected discovery result: %+v", result)
	}
	skill := result.Skills[0]
	if !skill.Policy.AllowImplicitInvocation {
		t.Fatalf("expected allow_implicit_invocation=true, got %+v", skill.Policy)
	}
	if got := strings.Join(skill.Dependencies.Tools, ","); got != "web_search,script_exec" {
		t.Fatalf("unexpected dependencies.tools: %q", got)
	}
}

func TestCatalogReturnsExplicitErrorsForInvalidSkillFiles(t *testing.T) {
	resetDiscoveryCacheForTests()
	root := t.TempDir()
	repoSkills := filepath.Join(root, ".agents", "skills")

	writeSkillFixture(t, filepath.Join(repoSkills, "invalid-frontmatter"), "", "missing name", "Body text.", "")
	writeSkillFixture(t, filepath.Join(repoSkills, "invalid-openai"), "broken", "Broken openai", "Body text.", "dependencies:\n  tools:\n    - 1\n")

	catalog := NewCatalogWithRoots([]string{repoSkills})
	result := catalog.ForceReload()
	if len(result.Skills) != 0 {
		t.Fatalf("expected no valid skills, got %+v", result.Skills)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected two explicit errors, got %+v", result.Errors)
	}
	if !containsError(result.Errors, "frontmatter field `name` is required") {
		t.Fatalf("missing frontmatter validation error: %+v", result.Errors)
	}
	if !containsError(result.Errors, "dependencies.tools") {
		t.Fatalf("missing openai dependency error: %+v", result.Errors)
	}
}

func TestCatalogCachesDiscoveryUntilForceReload(t *testing.T) {
	resetDiscoveryCacheForTests()
	root := t.TempDir()
	repoSkills := filepath.Join(root, ".agents", "skills")
	writeSkillFixture(t, filepath.Join(repoSkills, "s1"), "skill_one", "Skill one", "Body one.", "")

	catalog := NewCatalogWithRoots([]string{repoSkills})
	first := catalog.Discover()
	if len(first.Skills) != 1 {
		t.Fatalf("unexpected first discover result: %+v", first)
	}

	writeSkillFixture(t, filepath.Join(repoSkills, "s2"), "skill_two", "Skill two", "Body two.", "")
	second := catalog.Discover()
	if len(second.Skills) != 1 {
		t.Fatalf("expected cached discover result, got %+v", second)
	}

	reloaded := catalog.ForceReload()
	if len(reloaded.Skills) != 2 {
		t.Fatalf("expected force reload to include new skill, got %+v", reloaded)
	}
}

func TestCatalogSearchMatchesNameAndDescription(t *testing.T) {
	resetDiscoveryCacheForTests()
	root := t.TempDir()
	repoSkills := filepath.Join(root, ".agents", "skills")

	writeSkillFixture(t, filepath.Join(repoSkills, "release"), "release", "Deploy release", "Ship a release.", "")
	writeSkillFixture(t, filepath.Join(repoSkills, "summarize"), "summarize", "Weekly report", "Summarize the work.", "")

	catalog := NewCatalogWithRoots([]string{repoSkills})
	result := catalog.Search("deploy")
	if len(result.Skills) != 1 || result.Skills[0].Name != "release" {
		t.Fatalf("unexpected search result: %+v", result.Skills)
	}
}

func writeSkillFixture(t *testing.T, dir string, name string, description string, body string, openAI string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillPath := filepath.Join(dir, skillFileName)
	frontmatter := "---\n"
	if strings.TrimSpace(name) != "" {
		frontmatter += "name: " + name + "\n"
	}
	if strings.TrimSpace(description) != "" {
		frontmatter += "description: " + description + "\n"
	}
	frontmatter += "---\n\n" + body + "\n"
	if err := os.WriteFile(skillPath, []byte(frontmatter), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if strings.TrimSpace(openAI) == "" {
		return
	}
	configPath := filepath.Join(dir, "agents", "openai.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir openai dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(openAI), 0o644); err != nil {
		t.Fatalf("write openai.yaml: %v", err)
	}
}

func containsError(errors []DiscoveryError, snippet string) bool {
	for _, item := range errors {
		if strings.Contains(item.Error, snippet) {
			return true
		}
	}
	return false
}
