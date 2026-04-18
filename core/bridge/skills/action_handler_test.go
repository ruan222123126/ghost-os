package skills

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListActionIncludesRepoAndUserSources(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	writeSkillMarkdown(t, filepath.Join(projectRoot, ".agents", "skills", "release"), "release", "repo skill")
	writeSkillMarkdown(t, filepath.Join(homeRoot, ".ghost-os", "skills", "release"), "release", "user skill")

	handler := newTestActionHandler(t, projectRoot)
	payload, code, err := handler.ExecuteListAction("trace-skill-list")
	if err != nil {
		t.Fatalf("execute list: %v", err)
	}
	if code != 200 {
		t.Fatalf("unexpected code: got %d want 200", code)
	}

	items, ok := payload.([]SkillPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %T", payload)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 skills, got %d: %#v", len(items), items)
	}

	seenSources := map[string]bool{}
	seenIDs := map[string]bool{}
	for _, item := range items {
		seenSources[item.Source] = true
		seenIDs[item.ID] = true
		if !strings.Contains(item.Path, ".agents/skills") && !strings.Contains(item.Path, ".ghost-os/skills") {
			t.Fatalf("unexpected skill path: %q", item.Path)
		}
	}
	if !seenSources[SkillSourceRepo] || !seenSources[SkillSourceUser] {
		t.Fatalf("missing expected sources: %#v", items)
	}
	if len(seenIDs) != 2 {
		t.Fatalf("expected unique ids per source: %#v", items)
	}
}

func TestExecuteDeleteActionDeletesDirectoryAndReturnsNotFoundOnRepeat(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	writeSkillMarkdown(t, filepath.Join(projectRoot, ".agents", "skills", "cleanup"), "cleanup", "cleanup skill")
	handler := newTestActionHandler(t, projectRoot)

	items := mustListSkills(t, handler)
	if len(items) != 1 {
		t.Fatalf("expected one skill, got %#v", items)
	}
	target := items[0]

	payload, code, err := handler.ExecuteDeleteAction(SkillIDParams{ID: target.ID}, "trace-skill-delete")
	if err != nil {
		t.Fatalf("delete skill: %v", err)
	}
	if code != 200 {
		t.Fatalf("unexpected delete code: got %d want 200", code)
	}
	deleted, ok := payload.(SkillDeleteResponse)
	if !ok || !deleted.Deleted || deleted.ID != target.ID {
		t.Fatalf("unexpected delete payload: %#v", payload)
	}
	if _, statErr := os.Stat(target.Path); !os.IsNotExist(statErr) {
		t.Fatalf("expected skill dir removed, stat err=%v", statErr)
	}

	_, repeatCode, repeatErr := handler.ExecuteDeleteAction(SkillIDParams{ID: target.ID}, "trace-skill-delete-repeat")
	if !errors.Is(repeatErr, ErrSkillNotFound) {
		t.Fatalf("expected not found on repeat delete, got %v", repeatErr)
	}
	if repeatCode != 404 {
		t.Fatalf("unexpected repeat delete code: got %d want 404", repeatCode)
	}
}

func TestExecuteDeleteActionRejectsInvalidSkillID(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)
	handler := newTestActionHandler(t, projectRoot)

	_, code, err := handler.ExecuteDeleteAction(SkillIDParams{ID: "not-a-managed-id"}, "trace-invalid-id")
	if !errors.Is(err, ErrInvalidSkillID) {
		t.Fatalf("expected invalid id error, got %v", err)
	}
	if code != 400 {
		t.Fatalf("unexpected status code: got %d want 400", code)
	}

	traversalID := encodeRawSkillIDForTest(SkillSourceRepo, "../escape")
	_, traversalCode, traversalErr := handler.ExecuteDeleteAction(SkillIDParams{ID: traversalID}, "trace-traversal")
	if !errors.Is(traversalErr, ErrSkillPathForbidden) {
		t.Fatalf("expected traversal rejection, got %v", traversalErr)
	}
	if traversalCode != 400 {
		t.Fatalf("unexpected traversal code: got %d want 400", traversalCode)
	}
}

func TestResolveSkillRootsAutoCreatesUserSkillRoot(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	userRoot := filepath.Join(homeRoot, ".ghost-os", "skills")
	if _, err := os.Stat(userRoot); !os.IsNotExist(err) {
		t.Fatalf("expected user skill root to start missing, err=%v", err)
	}

	handler := newTestActionHandler(t, projectRoot)
	roots, err := handler.resolveSkillRoots()
	if err != nil {
		t.Fatalf("resolve skill roots: %v", err)
	}
	if roots.User != userRoot {
		t.Fatalf("unexpected user root: got %q want %q", roots.User, userRoot)
	}
	info, statErr := os.Stat(userRoot)
	if statErr != nil {
		t.Fatalf("expected user skill root created, stat err=%v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected user skill root directory, got file at %q", userRoot)
	}
}

func TestManagedSkillFromDiscoveryRejectsOutsideWhitelistRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	item := Skill{
		Name:        "escape",
		Description: "outside root",
		Path:        filepath.Join(outside, "escape", "SKILL.md"),
	}

	_, err := ManagedSkillFromDiscovery(SkillSourceRepo, root, item)
	if !errors.Is(err, ErrSkillPathForbidden) {
		t.Fatalf("expected outside root to be rejected, got %v", err)
	}
}

func mustListSkills(t *testing.T, handler *ActionHandler) []SkillPayload {
	t.Helper()
	payload, code, err := handler.ExecuteListAction("trace-list")
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if code != 200 {
		t.Fatalf("unexpected list code: got %d want 200", code)
	}
	items, ok := payload.([]SkillPayload)
	if !ok {
		t.Fatalf("unexpected list payload type: %T", payload)
	}
	return items
}

func writeSkillMarkdown(t *testing.T, dir string, name string, description string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nSkill body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func encodeRawSkillIDForTest(source string, relativePath string) string {
	raw := source + SkillIDSeparator + relativePath
	return SkillIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(raw))
}

type testStore struct {
	projectRoot string
}

func (s testStore) Config() (Config, error) {
	return Config{ProjectRoot: s.projectRoot}, nil
}

func newTestActionHandler(t *testing.T, projectRoot string) *ActionHandler {
	t.Helper()
	return NewActionHandler(&testStore{projectRoot: projectRoot}, nil)
}
