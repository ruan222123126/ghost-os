package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/session"
)

func TestSendFileToolExecuteExportsAndPersistsArtifactMetadata(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}

	var capturedParams map[string]any
	tool := NewSendFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "EXPORT_FILE" {
				t.Fatalf("unexpected action: got %q want %q", action, "EXPORT_FILE")
			}
			if traceID != "trace-send-file" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-send-file")
			}
			capturedParams = params
			return map[string]any{
				"artifact_id":   params["artifact_id"],
				"filename":      "report.txt",
				"mime_type":     "text/plain",
				"bytes":         12,
				"sha256":        "abc123",
				"stored_path":   filepath.Join(store.BaseDir(), "sessions", "session-1", "artifact-1.txt"),
				"original_path": "/tmp/report.txt",
			}, nil
		},
	}, store)

	ctx := WithToolCallID(
		WithSession(context.Background(), &session.Session{ID: "session-1"}),
		"call-send-file-1",
	)
	output, err := tool.Execute(
		ctx,
		json.RawMessage(`{"path":"/tmp/report.txt","title":"Quarterly Report","note":"share with the client"}`),
		"trace-send-file",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Contains(output, "stored_path") {
		t.Fatalf("send_file output should not include stored_path, got %q", output)
	}
	if capturedParams["session_id"] != "session-1" {
		t.Fatalf("unexpected session_id: %+v", capturedParams)
	}
	if capturedParams["artifact_root"] != store.BaseDir() {
		t.Fatalf("unexpected artifact_root: %+v", capturedParams)
	}

	result, ok := artifacts.DecodeSendFileResult(output)
	if !ok {
		t.Fatalf("expected decodable send_file payload, got %q", output)
	}
	if result.Artifact.Name != "Quarterly Report.txt" {
		t.Fatalf("unexpected display name: got %q want %q", result.Artifact.Name, "Quarterly Report.txt")
	}
	if result.Artifact.DownloadURL != "/api/sessions/session-1/artifacts/"+result.Artifact.ArtifactID {
		t.Fatalf("unexpected download URL: %q", result.Artifact.DownloadURL)
	}
	if !strings.Contains(result.Message, "Sent file:") {
		t.Fatalf("unexpected message: %q", result.Message)
	}

	loaded, err := store.Load("session-1", result.Artifact.ArtifactID)
	if err != nil {
		t.Fatalf("load persisted metadata: %v", err)
	}
	if loaded.Note != "share with the client" {
		t.Fatalf("unexpected note: got %q want %q", loaded.Note, "share with the client")
	}
}

func TestSendFileToolExecuteRequiresSessionContext(t *testing.T) {
	store, err := artifacts.NewSessionArtifactStore(t.TempDir())
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	tool := NewSendFileTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution client should not be called")
			return nil, nil
		},
	}, store)

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"/tmp/report.txt"}`), "trace-send-file"); err == nil {
		t.Fatal("expected missing session context error")
	}
}
