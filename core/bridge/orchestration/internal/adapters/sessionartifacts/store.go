package sessionartifacts

import (
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"ghost-os/bridge/artifacts"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
)

type Store struct {
	inner *artifacts.SessionArtifactStore
}

func NewFromEnv() (Store, error) {
	store, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		return Store{}, err
	}
	return Store{inner: store}, nil
}

func (s Store) Open(sessionID string, artifactID string) (appsessions.ArtifactDownload, error) {
	if s.inner == nil {
		return appsessions.ArtifactDownload{}, appsessions.ErrArtifactStoreRequired
	}
	normalizedSessionID, err := artifacts.NormalizeSessionID(sessionID)
	if err != nil {
		return appsessions.ArtifactDownload{}, appsessions.WrapInvalidArtifactSessionID(err)
	}
	normalizedArtifactID, err := artifacts.NormalizeArtifactID(artifactID)
	if err != nil {
		return appsessions.ArtifactDownload{}, appsessions.WrapInvalidArtifactID(err)
	}

	file, info, artifact, err := s.inner.OpenStoredFile(normalizedSessionID, normalizedArtifactID)
	if err != nil {
		return appsessions.ArtifactDownload{}, mapOpenError(err)
	}
	return appsessions.ArtifactDownload{
		Reader:   file,
		Name:     artifact.Name,
		MimeType: resolveDownloadMimeType(artifact.Name, artifact.MimeType),
		Size:     info.Size(),
		SHA256:   artifact.SHA256,
	}, nil
}

func mapOpenError(err error) error {
	switch {
	case errors.Is(err, artifacts.ErrArtifactNotFound), errors.Is(err, os.ErrNotExist):
		return appsessions.WrapArtifactNotFound(err)
	case errors.Is(err, artifacts.ErrInvalidStoredPath):
		return appsessions.WrapInvalidArtifactPath(err)
	default:
		return err
	}
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
