package transport

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type skillResponsePayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
}

func TestHandleSkillsListReturnsSourcesAndSupportsEmptyList(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	handler := newTestHandler(t, nil)
	empty := serveRequest(handler, http.MethodGet, "/api/skills", "", nil)
	if empty.Code != http.StatusOK {
		t.Fatalf("unexpected empty-list status: got %d want %d", empty.Code, http.StatusOK)
	}
	if items := decodeSkillPayloadList(t, empty); len(items) != 0 {
		t.Fatalf("expected empty list, got %#v", items)
	}

	writeSkillFixture(t, filepath.Join(projectRoot, ".agents", "skills", "repo_one"), "repo_one", "repo skill")
	writeSkillFixture(t, filepath.Join(homeRoot, ".ghost-os", "skills", "user_one"), "user_one", "user skill")

	withData := serveRequest(handler, http.MethodGet, "/api/skills", "", nil)
	if withData.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", withData.Code, http.StatusOK)
	}
	items := decodeSkillPayloadList(t, withData)
	if len(items) != 2 {
		t.Fatalf("expected 2 skills, got %#v", items)
	}
	if !hasSkillSource(items, "repo") || !hasSkillSource(items, "user") {
		t.Fatalf("expected repo+user sources, got %#v", items)
	}
	if !items[0].Enabled || !items[1].Enabled {
		t.Fatalf("expected enabled=true by default, got %#v", items)
	}
}

func TestHandleSkillPatchUpdatesEnabledState(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	writeSkillFixture(t, filepath.Join(projectRoot, ".agents", "skills", "release"), "release", "repo skill")
	handler := newTestHandler(t, nil)

	list := serveRequest(handler, http.MethodGet, "/api/skills", "", nil)
	items := decodeSkillPayloadList(t, list)
	if len(items) != 1 {
		t.Fatalf("expected one skill, got %#v", items)
	}

	patch := serveRequest(handler, http.MethodPatch, "/api/skills/"+items[0].ID, `{"enabled":false}`, nil)
	if patch.Code != http.StatusOK {
		t.Fatalf("unexpected patch status: got %d want %d body=%s", patch.Code, http.StatusOK, patch.Body.String())
	}
	updated := decodeSingleSkillPayload(t, patch)
	if updated.Enabled {
		t.Fatalf("expected disabled payload, got %#v", updated)
	}

	refresh := serveRequest(handler, http.MethodGet, "/api/skills", "", nil)
	refreshed := decodeSkillPayloadList(t, refresh)
	if len(refreshed) != 1 || refreshed[0].Enabled {
		t.Fatalf("expected persisted disabled state, got %#v", refreshed)
	}
}

func TestHandleSkillDeleteSupportsSuccessRepeatDeleteAndInvalidID(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)

	writeSkillFixture(t, filepath.Join(projectRoot, ".agents", "skills", "cleanup"), "cleanup", "cleanup skill")
	handler := newTestHandler(t, nil)

	list := serveRequest(handler, http.MethodGet, "/api/skills", "", nil)
	items := decodeSkillPayloadList(t, list)
	if len(items) != 1 {
		t.Fatalf("expected one skill, got %#v", items)
	}
	id := items[0].ID

	firstDelete := serveRequest(handler, http.MethodDelete, "/api/skills/"+id, "", nil)
	if firstDelete.Code != http.StatusOK {
		t.Fatalf("unexpected first delete status: got %d want %d", firstDelete.Code, http.StatusOK)
	}

	repeatDelete := serveRequest(handler, http.MethodDelete, "/api/skills/"+id, "", nil)
	if repeatDelete.Code != http.StatusNotFound {
		t.Fatalf("unexpected repeat delete status: got %d want %d", repeatDelete.Code, http.StatusNotFound)
	}

	invalid := serveRequest(handler, http.MethodDelete, "/api/skills/not-a-managed-id", "", nil)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("unexpected invalid id status: got %d want %d", invalid.Code, http.StatusBadRequest)
	}
}

func TestHandleSkillDeleteRejectsPathTraversalID(t *testing.T) {
	projectRoot := t.TempDir()
	homeRoot := t.TempDir()
	t.Setenv("GHOST_PROJECT_ROOT", projectRoot)
	t.Setenv("HOME", homeRoot)
	handler := newTestHandler(t, nil)

	traversalID := encodeSkillIDForTransportTest("repo", "../escape")
	recorder := serveRequest(handler, http.MethodDelete, "/api/skills/"+traversalID, "", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected traversal status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
}

func decodeSkillPayloadList(t *testing.T, recorder *httptest.ResponseRecorder) []skillResponsePayload {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload []skillResponsePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}

func decodeSingleSkillPayload(t *testing.T, recorder *httptest.ResponseRecorder) skillResponsePayload {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload skillResponsePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}

func hasSkillSource(items []skillResponsePayload, source string) bool {
	for _, item := range items {
		if item.Source == source {
			return true
		}
	}
	return false
}

func writeSkillFixture(t *testing.T, dir string, name string, description string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nSkill body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func encodeSkillIDForTransportTest(source string, relativePath string) string {
	raw := source + "|" + relativePath
	return "skill_" + base64.RawURLEncoding.EncodeToString([]byte(raw))
}
