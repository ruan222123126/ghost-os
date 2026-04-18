package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteFindIconTemplateUploadStoresTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	payload, err := executeFindIconTemplateUpload(findIconTemplateUploadRequest{
		Filename: "icon.png",
		MimeType: "image/png",
		DataURL:  "data:image/png;base64,R2hvc3Q=",
	})
	if err != nil {
		t.Fatalf("executeFindIconTemplateUpload returned error: %v", err)
	}
	if !strings.HasPrefix(payload.TemplatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", payload.TemplatePath)
	}
	if len(payload.SHA256) != 64 {
		t.Fatalf("unexpected sha256 length: %q", payload.SHA256)
	}
	if _, err := os.Stat(payload.TemplatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
}

func TestNormalizeFindIconPreviewRequestRejectsInvalidThreshold(t *testing.T) {
	_, err := normalizeFindIconPreviewRequest(findIconPreviewRequest{
		TemplatePath: "/tmp/icon.png",
		Threshold:    floatPtr(1.1),
	})
	if err == nil || !strings.Contains(err.Error(), "threshold must be between 0 and 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeFindIconPreviewPayloadParsesMatches(t *testing.T) {
	payload, err := decodeFindIconPreviewPayload(`{"display_id":1,"matches":[{"score":0.95}]}`)
	if err != nil {
		t.Fatalf("decodeFindIconPreviewPayload returned error: %v", err)
	}
	if !payload.Exists || payload.MatchCount != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.DisplayID == nil || *payload.DisplayID != 1 {
		t.Fatalf("unexpected display id: %+v", payload.DisplayID)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
