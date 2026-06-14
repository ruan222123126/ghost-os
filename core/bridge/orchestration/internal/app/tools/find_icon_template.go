package tools

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

const (
	FindIconTemplateUploadAction = "FIND_ICON_TEMPLATE_UPLOAD"
	FindIconPreviewAction        = "FIND_ICON_PREVIEW"
	FindIconTemplateRoot         = "~/.ghost-os/screen_templates/workflow"
	ScreenControlToolID          = "screen_control"
)

func ExecuteFindIconTemplateUpload(req api.FindIconTemplateUploadRequest) (api.FindIconTemplateUploadPayload, error) {
	filename, mimeType, dataURL, err := NormalizeFindIconTemplateUpload(req)
	if err != nil {
		return api.FindIconTemplateUploadPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	decodedBytes, dataMimeType, err := DecodeFindIconTemplateDataURL(dataURL)
	if err != nil {
		return api.FindIconTemplateUploadPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	if err := ValidateFindIconMimeTypeConsistency(mimeType, dataMimeType); err != nil {
		return api.FindIconTemplateUploadPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	ext, err := ResolveFindIconTemplateExtension(filename, mimeType, dataMimeType)
	if err != nil {
		return api.FindIconTemplateUploadPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	path, sha, err := StoreFindIconTemplateFile(decodedBytes, ext)
	if err != nil {
		return api.FindIconTemplateUploadPayload{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	return api.FindIconTemplateUploadPayload{
		TemplatePath: path,
		TemplateName: filepath.Base(path),
		SHA256:       sha,
	}, nil
}

func NormalizeFindIconTemplateUpload(req api.FindIconTemplateUploadRequest) (string, string, string, error) {
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		return "", "", "", fmt.Errorf("filename is required")
	}
	mimeType := strings.ToLower(strings.TrimSpace(req.MimeType))
	if !strings.HasPrefix(mimeType, "image/") {
		return "", "", "", fmt.Errorf("mime_type must be image/*")
	}
	dataURL := strings.TrimSpace(req.DataURL)
	if dataURL == "" {
		return "", "", "", fmt.Errorf("data_url is required")
	}
	return filename, mimeType, dataURL, nil
}

func DecodeFindIconTemplateDataURL(raw string) ([]byte, string, error) {
	if !strings.HasPrefix(raw, "data:") {
		return nil, "", errors.New("data_url must start with data: URI scheme")
	}
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("data_url is invalid")
	}
	return decodeFindIconTemplateDataURLParts(parts)
}

func decodeFindIconTemplateDataURLParts(parts []string) ([]byte, string, error) {
	meta := strings.TrimPrefix(parts[0], "data:")
	if !strings.HasSuffix(meta, ";base64") {
		return nil, "", fmt.Errorf("data_url must be base64 encoded")
	}
	mimeType := strings.ToLower(strings.TrimSuffix(meta, ";base64"))
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("data_url mime must be image/*")
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("decode data_url base64: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("template image is empty")
	}
	return data, mimeType, nil
}

func ValidateFindIconMimeTypeConsistency(reqMime string, dataMime string) error {
	if strings.TrimSpace(reqMime) == "" || strings.TrimSpace(dataMime) == "" {
		return fmt.Errorf("template mime type is missing")
	}
	if reqMime != dataMime {
		return fmt.Errorf("mime_type and data_url mime do not match")
	}
	return nil
}

func ResolveFindIconTemplateExtension(filename string, mimeTypes ...string) (string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if IsAllowedFindIconTemplateExt(ext) {
		return ext, nil
	}
	for _, item := range mimeTypes {
		if resolved, ok := findIconTemplateExtByMIME[strings.ToLower(strings.TrimSpace(item))]; ok {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("unsupported template image type")
}

var findIconTemplateExtByMIME = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/webp": ".webp",
	"image/bmp":  ".bmp",
}

func IsAllowedFindIconTemplateExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func StoreFindIconTemplateFile(data []byte, ext string) (string, string, error) {
	root, err := ResolveFindIconTemplateRoot()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", fmt.Errorf("create template root: %w", err)
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	path := filepath.Join(root, sha+NormalizeFindIconExt(ext))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", "", fmt.Errorf("write template file: %w", err)
	}
	return path, sha, nil
}

func NormalizeFindIconExt(ext string) string {
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func ResolveFindIconTemplateRoot() (string, error) {
	path := strings.TrimSpace(FindIconTemplateRoot)
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		}
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve template root path: %w", err)
	}
	return filepath.Clean(absPath), nil
}
