package transport

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestConfigEndpointReadsAndClearsProjectRoot(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	projectRoot := filepath.Join(homeDir, "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", projectRoot, err)
	}
	handler := newTestHandler(t, nil)

	update := serveRequest(
		handler,
		http.MethodPost,
		"/api/config",
		`{"project_root":"~/repo"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}
	updateBody := decodeResponseBody(t, update)
	updatePayload := updateBody.Payload.(map[string]any)
	if updatePayload["project_root"] != projectRoot {
		t.Fatalf("unexpected updated project_root: got %v want %q", updatePayload["project_root"], projectRoot)
	}

	reset := serveRequest(
		handler,
		http.MethodPost,
		"/api/config",
		`{"project_root":""}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if reset.Code != http.StatusOK {
		t.Fatalf("unexpected reset status: got %d want %d", reset.Code, http.StatusOK)
	}
	resetBody := decodeResponseBody(t, reset)
	resetPayload := resetBody.Payload.(map[string]any)
	if resetPayload["project_root"] != "" {
		t.Fatalf("unexpected reset project_root: got %v want empty string", resetPayload["project_root"])
	}
}

func TestConfigEndpointRejectsInvalidProjectRoot(t *testing.T) {
	handler := newTestHandler(t, nil)
	filePath := filepath.Join(t.TempDir(), "plain-file")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", filePath, err)
	}

	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "relative", body: `{"project_root":"relative/path"}`, want: "project_root must be an absolute path"},
		{name: "missing", body: `{"project_root":"/definitely/missing/path"}`, want: "project_root does not exist"},
		{name: "file", body: `{"project_root":"` + filePath + `"}`, want: "project_root must be a directory"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := serveRequest(
				handler,
				http.MethodPost,
				"/api/config",
				tc.body,
				map[string]string{"Content-Type": "application/json"},
			)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
			}
			if !strings.Contains(recorder.Body.String(), tc.want) {
				t.Fatalf("unexpected body: %s", recorder.Body.String())
			}
		})
	}
}

func TestAgentEndpointAppliesRequestProjectRootWithoutPersisting(t *testing.T) {
	projectRoot := t.TempDir()
	var capturedRoot string
	handler, service, _ := newTestHandlerWithService(t, func(
		_ context.Context,
		_ string,
		_ string,
		_ string,
		store bridgeconfig.Store,
		_ *session.Store,
	) (string, string, error) {
		cfg, err := store.Config()
		if err != nil {
			return "", "", err
		}
		capturedRoot = cfg.ProjectRoot
		return "ok", "session-project-root", nil
	}, nil)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"hello","project_root":"`+projectRoot+`"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if capturedRoot != projectRoot {
		t.Fatalf("unexpected executor project_root: got %q want %q", capturedRoot, projectRoot)
	}

	assertProjectRootNotPersisted(t, service)
}

func TestAgentStreamEndpointAppliesRequestProjectRootWithoutPersisting(t *testing.T) {
	projectRoot := t.TempDir()
	var capturedRoot string
	streamExecutor := func(
		ctx context.Context,
		_ string,
		_ string,
		traceID string,
		store bridgeconfig.Store,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		cfg, err := store.Config()
		if err != nil {
			return "", "", err
		}
		capturedRoot = cfg.ProjectRoot
		sessionID := "session-project-root-stream"
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 0, "", streaming.EventRunStarted, map[string]any{
			"session_id": sessionID,
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, map[string]any{
			"text":       "ok",
			"session_id": sessionID,
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 1, "", streaming.EventDone, map[string]any{
			"session_id":    sessionID,
			"session_ended": false,
		})); err != nil {
			return "", "", err
		}
		return "ok", sessionID, nil
	}
	handler, service, _ := newTestHandlerWithService(t, nil, streamExecutor)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","project_root":"`+projectRoot+`","trace_id":"trace-project-root-stream"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if capturedRoot != projectRoot {
		t.Fatalf("unexpected stream executor project_root: got %q want %q", capturedRoot, projectRoot)
	}

	assertProjectRootNotPersisted(t, service)
}

func assertProjectRootNotPersisted(t *testing.T, service *bridgeorchestration.Service) {
	t.Helper()

	snapshot, err := service.ConfigStore().PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ProjectRoot != "" {
		t.Fatalf("expected request override to stay non-persistent, got %q", snapshot.ProjectRoot)
	}

	configPath := os.Getenv("GHOST_CONFIG_PATH")
	body, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", configPath, err)
	}
	if strings.Contains(string(body), "project_root") {
		t.Fatalf("expected config file to stay unchanged, got %s", string(body))
	}
}
