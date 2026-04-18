package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ghost-os/bridge/tools"
)

const (
	findIconTemplateUploadAction = "FIND_ICON_TEMPLATE_UPLOAD"
	findIconPreviewAction        = "FIND_ICON_PREVIEW"
	findIconTemplateRoot         = "~/.ghost-os/screen_templates/workflow"
	screenControlToolID          = "screen_control"
)

func (s *bridgeService) executeFindIconTemplateUploadActionResult(
	req findIconTemplateUploadRequest,
	traceID string,
) (ServiceResult, error) {
	payload, err := executeFindIconTemplateUpload(req)
	if err != nil {
		logAction(traceID, findIconTemplateUploadAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconTemplateUploadAction, "success", nil)
	return serviceResultCreated(payload), nil
}

func executeFindIconTemplateUpload(req findIconTemplateUploadRequest) (findIconTemplateUploadPayload, error) {
	filename, mimeType, dataURL, err := normalizeFindIconTemplateUpload(req)
	if err != nil {
		return findIconTemplateUploadPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	decodedBytes, dataMimeType, err := decodeFindIconTemplateDataURL(dataURL)
	if err != nil {
		return findIconTemplateUploadPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	if err := validateFindIconMimeTypeConsistency(mimeType, dataMimeType); err != nil {
		return findIconTemplateUploadPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	ext, err := resolveFindIconTemplateExtension(filename, mimeType, dataMimeType)
	if err != nil {
		return findIconTemplateUploadPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	path, sha, err := storeFindIconTemplateFile(decodedBytes, ext)
	if err != nil {
		return findIconTemplateUploadPayload{}, wrapServiceError(ServiceErrorInternal, err)
	}
	return findIconTemplateUploadPayload{
		TemplatePath: path,
		TemplateName: filepath.Base(path),
		SHA256:       sha,
	}, nil
}

func normalizeFindIconTemplateUpload(req findIconTemplateUploadRequest) (string, string, string, error) {
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

func decodeFindIconTemplateDataURL(raw string) ([]byte, string, error) {
	if !strings.HasPrefix(raw, "data:") {
		return nil, "", fmt.Errorf("data_url must start with data:")
	}
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("data_url is invalid")
	}
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

func validateFindIconMimeTypeConsistency(reqMime string, dataMime string) error {
	if strings.TrimSpace(reqMime) == "" || strings.TrimSpace(dataMime) == "" {
		return fmt.Errorf("template mime type is missing")
	}
	if reqMime != dataMime {
		return fmt.Errorf("mime_type and data_url mime do not match")
	}
	return nil
}

func resolveFindIconTemplateExtension(filename string, mimeTypes ...string) (string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if isAllowedFindIconTemplateExt(ext) {
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

func isAllowedFindIconTemplateExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func storeFindIconTemplateFile(data []byte, ext string) (string, string, error) {
	root, err := resolveFindIconTemplateRoot()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", fmt.Errorf("create template root: %w", err)
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	path := filepath.Join(root, sha+normalizeFindIconExt(ext))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", "", fmt.Errorf("write template file: %w", err)
	}
	return path, sha, nil
}

func normalizeFindIconExt(ext string) string {
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

func resolveFindIconTemplateRoot() (string, error) {
	path := strings.TrimSpace(findIconTemplateRoot)
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

func (s *bridgeService) executeFindIconPreviewActionResult(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (ServiceResult, error) {
	params, err := normalizeFindIconPreviewRequest(req)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	payload, err := s.runFindIconPreview(ctx, params, traceID)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconPreviewAction, "success", nil)
	return serviceResultSuccess(payload), nil
}

func normalizeFindIconPreviewRequest(req findIconPreviewRequest) (findIconPreviewRequest, error) {
	templatePath := strings.TrimSpace(req.TemplatePath)
	if templatePath == "" {
		return findIconPreviewRequest{}, fmt.Errorf("template_path is required")
	}
	out := req
	out.TemplatePath = templatePath
	if req.Threshold != nil && (*req.Threshold < 0 || *req.Threshold > 1) {
		return findIconPreviewRequest{}, fmt.Errorf("threshold must be between 0 and 1")
	}
	if req.MaxResults != nil && *req.MaxResults < 1 {
		return findIconPreviewRequest{}, fmt.Errorf("max_results must be >= 1")
	}
	if req.DisplayID != nil && *req.DisplayID < 0 {
		return findIconPreviewRequest{}, fmt.Errorf("display_id must be >= 0")
	}
	return out, nil
}

func (s *bridgeService) runFindIconPreview(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	if s == nil || s.runtimeFactory == nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("runtime factory is not configured"))
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("build runtime dependencies: %w", err))
	}
	defer deps.Close()
	if deps.registry == nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool registry is not configured"))
	}
	tool := deps.registry.Get(screenControlToolID)
	if tool == nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool %q is not available", screenControlToolID))
	}
	return executeFindIconPreviewToolCall(ctx, tool, req, traceID)
}

func executeFindIconPreviewToolCall(
	ctx context.Context,
	tool tools.Tool,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	args, err := json.Marshal(buildFindIconPreviewToolArgs(req))
	if err != nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorInternal, fmt.Errorf("encode find_icon args: %w", err))
	}
	output, err := tool.Execute(tools.WithToolCallID(ctx, "find-icon-preview"), args, traceID)
	if err != nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	payload, err := decodeFindIconPreviewPayload(output)
	if err != nil {
		return findIconPreviewPayload{}, wrapServiceError(ServiceErrorInternal, err)
	}
	if req.HoverAfterMatch && payload.Exists {
		if err := executeFindIconPreviewHover(ctx, tool, payload, traceID); err != nil {
			return findIconPreviewPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		payload.Hovered = true
	}
	return payload, nil
}
