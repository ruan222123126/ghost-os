package downloads

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type BinaryDownload struct {
	Reader   io.ReadCloser
	Name     string
	MimeType string
	Size     int64
	SHA256   string
}

func OpenFindIconTemplate(templatePath string) (BinaryDownload, error) {
	path, err := normalizeFindIconTemplateDownloadPath(templatePath)
	if err != nil {
		return BinaryDownload{}, err
	}
	file, info, err := openRegularDownloadFile(path)
	if err != nil {
		return BinaryDownload{}, err
	}
	return BinaryDownload{
		Reader:   file,
		Name:     filepath.Base(path),
		MimeType: resolveDownloadMimeType(path, ""),
		Size:     info.Size(),
	}, nil
}

func normalizeFindIconTemplateDownloadPath(templatePath string) (string, error) {
	trimmed := strings.TrimSpace(templatePath)
	if trimmed == "" {
		return "", bus.WrapError(bus.ServiceErrorInvalidInput, errors.New("template_path is required"))
	}
	absolutePath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", bus.WrapError(bus.ServiceErrorInvalidInput, errors.New("template_path is invalid"))
	}
	rootPath, err := apptools.ResolveFindIconTemplateRoot()
	if err != nil {
		return "", bus.WrapError(bus.ServiceErrorInternal, fmt.Errorf("resolve find_icon template root: %w", err))
	}
	cleanPath := filepath.Clean(absolutePath)
	withinRoot, err := isPathWithinRoot(cleanPath, rootPath)
	if err != nil {
		return "", bus.WrapError(bus.ServiceErrorInvalidInput, errors.New("template_path is invalid"))
	}
	if !withinRoot {
		return "", bus.WrapError(
			bus.ServiceErrorInvalidInput,
			errors.New("template_path must be inside workflow template root"),
		)
	}
	return cleanPath, nil
}

func openRegularDownloadFile(path string) (*os.File, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, bus.WrapError(bus.ServiceErrorNotFound, errors.New("template image not found"))
		}
		return nil, nil, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, nil, bus.WrapError(
			bus.ServiceErrorInvalidInput,
			errors.New("template_path must point to an image file"),
		)
	}
	return file, info, nil
}

func isPathWithinRoot(path string, root string) (bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	if rel == ".." {
		return false, nil
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func resolveDownloadMimeType(name string, configured string) string {
	if mimeType := strings.TrimSpace(configured); mimeType != "" {
		return mimeType
	}
	byExtension := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}
