package sessions

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

const ActionDownloadArtifact = "SESSION_ARTIFACT_DOWNLOAD"

var (
	ErrArtifactStoreRequired    = errors.New("artifact store is not configured")
	ErrArtifactIDRequired       = errors.New("artifact id is required")
	ErrInvalidArtifactSessionID = errors.New("invalid artifact session id")
	ErrInvalidArtifactID        = errors.New("invalid artifact id")
	ErrArtifactNotFound         = errors.New("artifact not found")
	ErrInvalidArtifactPath      = errors.New("invalid artifact path")
)

type ArtifactDownload struct {
	Reader   io.ReadCloser
	Name     string
	MimeType string
	Size     int64
	SHA256   string
}

type ArtifactStore interface {
	Open(sessionID string, artifactID string) (ArtifactDownload, error)
}

func (s Service) DownloadArtifact(
	sessionID string,
	artifactID string,
	traceID string,
) (ArtifactDownload, error) {
	id, err := RequireSessionID(sessionID)
	if err != nil {
		s.log(traceID, ActionDownloadArtifact, "error", err)
		return ArtifactDownload{}, err
	}
	artifact, err := requireArtifactID(artifactID)
	if err != nil {
		s.log(traceID, ActionDownloadArtifact, "error", err)
		return ArtifactDownload{}, err
	}
	store, err := s.requireArtifactStore()
	if err != nil {
		s.log(traceID, ActionDownloadArtifact, "error", err)
		return ArtifactDownload{}, err
	}

	download, err := store.Open(id, artifact)
	if err != nil {
		s.log(traceID, ActionDownloadArtifact, "error", err)
		return ArtifactDownload{}, err
	}
	s.log(traceID, ActionDownloadArtifact, "success", nil)
	return download, nil
}

func requireArtifactID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", ErrArtifactIDRequired
	}
	return trimmed, nil
}

func (s Service) requireArtifactStore() (ArtifactStore, error) {
	if s.ArtifactStore == nil {
		return nil, ErrArtifactStoreRequired
	}
	return s.ArtifactStore, nil
}

func WrapInvalidArtifactID(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrInvalidArtifactID, err)
}

func WrapInvalidArtifactSessionID(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrInvalidArtifactSessionID, err)
}

func WrapArtifactNotFound(err error) error {
	if err == nil {
		return ErrArtifactNotFound
	}
	return fmt.Errorf("%w: %v", ErrArtifactNotFound, err)
}

func WrapInvalidArtifactPath(err error) error {
	if err == nil {
		return ErrInvalidArtifactPath
	}
	return fmt.Errorf("%w: %v", ErrInvalidArtifactPath, err)
}
