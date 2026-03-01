package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveImageSourceRejectsOversizedLocalFile(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "oversized.png")

	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatalf("create temp image: %v", err)
	}
	if err := file.Truncate(maxInlineImageBytes + 1); err != nil {
		file.Close()
		t.Fatalf("truncate temp image: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp image: %v", err)
	}

	_, err = resolveImageSource(&ImageContent{Path: imagePath})
	if err == nil {
		t.Fatal("expected size limit error but got nil")
	}
	if !strings.Contains(err.Error(), "exceeds max inline size") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOpenAIImageURLBuildsDataURLFromLocalFile(t *testing.T) {
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "tool-shot.png")
	if err := os.WriteFile(imagePath, []byte("fake-image"), 0o600); err != nil {
		t.Fatalf("write temp image: %v", err)
	}

	url, err := resolveOpenAIImageURL(&ImageContent{
		Path:     imagePath,
		MimeType: "image/png",
	})
	if err != nil {
		t.Fatalf("resolveOpenAIImageURL returned error: %v", err)
	}
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("unexpected data url: %s", url)
	}
}
