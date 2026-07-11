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
		if source, handled, err := parseInlineDataURL(url, image.MimeType); handled {
			return source, err
		}
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

func parseInlineDataURL(rawURL string, explicitMime string) (imageSource, bool, error) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(rawURL)), "data:") {
		return imageSource{}, false, nil
	}

	mediaType, data, err := decodeInlineDataURL(rawURL, explicitMime)
	if err != nil {
		return imageSource{}, true, err
	}
	return imageSource{
		MediaType:  mediaType,
		Base64Data: data,
	}, true, nil
}

func decodeInlineDataURL(rawURL string, explicitMime string) (string, string, error) {
	body := strings.TrimSpace(rawURL[5:])
	header, payload, ok := strings.Cut(body, ",")
	if !ok {
		return "", "", fmt.Errorf("invalid data url: missing payload")
	}

	mediaType, isBase64 := parseInlineDataHeader(header, explicitMime)
	if !isBase64 {
		return "", "", fmt.Errorf("invalid data url: image data must be base64-encoded")
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", "", fmt.Errorf("invalid data url: payload is empty")
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return "", "", fmt.Errorf("invalid data url: decode base64 payload: %w", err)
	}
	return mediaType, payload, nil
}

func parseInlineDataHeader(header string, explicitMime string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(header), ";")
	mediaType := resolveImageMimeType(explicitMime, "")
	if len(parts) > 0 {
		if candidate := strings.TrimSpace(parts[0]); candidate != "" {
			mediaType = candidate
		}
	}
	for _, part := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			return mediaType, true
		}
	}
	return mediaType, false
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
