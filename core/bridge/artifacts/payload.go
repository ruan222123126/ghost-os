package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SendFileResult struct {
	Message  string                    `json:"message"`
	Artifact PublicSessionFileArtifact `json:"artifact"`
}

// PublicSessionFileArtifact is the outward-facing artifact payload used in tool outputs
// and session projections. It intentionally omits internal-only fields like StoredPath
// to keep session history portable across environments.
type PublicSessionFileArtifact struct {
	ArtifactID  string `json:"artifact_id"`
	Name        string `json:"name"`
	MimeType    string `json:"mime_type,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	DownloadURL string `json:"download_url"`
	SourcePath  string `json:"source_path,omitempty"`
	Note        string `json:"note,omitempty"`
}

func PublicSessionFileArtifactFromMetadata(artifact SessionFileArtifact) PublicSessionFileArtifact {
	return PublicSessionFileArtifact{
		ArtifactID:  strings.TrimSpace(artifact.ArtifactID),
		Name:        strings.TrimSpace(artifact.Name),
		MimeType:    strings.TrimSpace(artifact.MimeType),
		Bytes:       artifact.Bytes,
		SHA256:      strings.TrimSpace(artifact.SHA256),
		DownloadURL: strings.TrimSpace(artifact.DownloadURL),
		SourcePath:  strings.TrimSpace(artifact.SourcePath),
		Note:        strings.TrimSpace(artifact.Note),
	}
}

func EncodeSendFileResult(result SendFileResult) (string, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode send_file result: %w", err)
	}
	return string(encoded), nil
}

func DecodeSendFileResult(raw string) (SendFileResult, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SendFileResult{}, false
	}
	var result SendFileResult
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return SendFileResult{}, false
	}
	if strings.TrimSpace(result.Artifact.ArtifactID) == "" || strings.TrimSpace(result.Artifact.DownloadURL) == "" {
		return SendFileResult{}, false
	}
	return result, true
}
