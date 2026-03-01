package llm

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxInlineImageBytes int64 = 20 << 20 // 20 MiB

type imageSource struct {
	URL        string
	MediaType  string
	Base64Data string
}

func (s imageSource) dataURL() string {
	return fmt.Sprintf("data:%s;base64,%s", s.MediaType, s.Base64Data)
}

func resolveImageSource(image *ImageContent) (imageSource, error) {
	if image == nil {
		return imageSource{}, fmt.Errorf("image content is nil")
	}

	if url := strings.TrimSpace(image.URL); url != "" {
		return imageSource{URL: url}, nil
	}

	path := strings.TrimSpace(image.Path)
	if path == "" {
		return imageSource{}, fmt.Errorf("image content requires path or url")
	}

	data, err := readLocalImageWithLimit(path, maxInlineImageBytes)
	if err != nil {
		return imageSource{}, err
	}
	return imageSource{
		MediaType:  resolveImageMimeType(image.MimeType, path),
		Base64Data: base64.StdEncoding.EncodeToString(data),
	}, nil
}

func readLocalImageWithLimit(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read image %q: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat image %q: %w", path, err)
	}
	if info.Mode().IsRegular() && info.Size() > maxBytes {
		return nil, fmt.Errorf("image %q exceeds max inline size %d bytes", path, maxBytes)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image %q: %w", path, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image %q exceeds max inline size %d bytes", path, maxBytes)
	}
	return data, nil
}

func resolveOpenAIImageURL(image *ImageContent) (string, error) {
	source, err := resolveImageSource(image)
	if err != nil {
		return "", err
	}
	if source.URL != "" {
		return source.URL, nil
	}
	return source.dataURL(), nil
}

func resolveImageMimeType(explicit string, path string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}

	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/png"
	}
}
