package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/tools/internal/payloadutil"
)

const defaultSendFileMaxBytes = 100 << 20

type SendFileTool struct {
	execution ExecutionClient
	store     *artifacts.SessionArtifactStore
}

type sendFileArgs struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
	Note  string `json:"note,omitempty"`
}

func NewSendFileTool(client ExecutionClient, store *artifacts.SessionArtifactStore) Tool {
	return &SendFileTool{execution: client, store: store}
}

func (SendFileTool) Name() string {
	return "send_file"
}

func (SendFileTool) Description() string {
	return "Send a local file back to the current session as a downloadable attachment."
}

func (SendFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"Absolute or workspace-relative path to the file to send back to the client."},
			"title":{"type":"string","description":"Optional display name shown in the client. Original extension is preserved when missing."},
			"note":{"type":"string","description":"Optional short note shown with the attachment."}
		},
		"required":["path"],
		"additionalProperties":false
	}`)
}

func (t *SendFileTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t == nil || t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	if t.store == nil {
		return "", fmt.Errorf("artifact store is not configured")
	}

	var args sendFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	sess := SessionFromContext(ctx)
	if sess == nil || strings.TrimSpace(sess.ID) == "" {
		return "", fmt.Errorf("send_file requires an active session")
	}

	artifactID := buildSendFileArtifactID(traceID, ToolCallIDFromContext(ctx))
	payload, err := t.execution.Call(ctx, "EXPORT_FILE", map[string]any{
		"path":          path,
		"session_id":    strings.TrimSpace(sess.ID),
		"artifact_root": t.store.BaseDir(),
		"artifact_id":   artifactID,
		"max_bytes":     defaultSendFileMaxBytes,
	}, traceID)
	if err != nil {
		return "", fmt.Errorf("execution EXPORT_FILE failed: %w", err)
	}

	filename, err := payloadutil.String(payload, "filename")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	storedPath, err := payloadutil.String(payload, "stored_path")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	originalPath, err := payloadutil.String(payload, "original_path")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	mimeType, err := payloadutil.String(payload, "mime_type")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	sha256, err := payloadutil.String(payload, "sha256")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	bytesCount, err := payloadutil.Int(payload, "bytes")
	if err != nil {
		return "", fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}

	displayName := normalizeSendFileName(strings.TrimSpace(args.Title), filename)
	artifact := artifacts.SessionFileArtifact{
		ArtifactID:  artifactID,
		SessionID:   strings.TrimSpace(sess.ID),
		Name:        displayName,
		MimeType:    strings.TrimSpace(mimeType),
		Bytes:       int64(bytesCount),
		SHA256:      strings.TrimSpace(sha256),
		DownloadURL: fmt.Sprintf("/api/sessions/%s/artifacts/%s", strings.TrimSpace(sess.ID), artifactID),
		SourcePath:  strings.TrimSpace(originalPath),
		StoredPath:  strings.TrimSpace(storedPath),
		Note:        strings.TrimSpace(args.Note),
	}
	if err := t.store.WriteMetadata(artifact); err != nil {
		return "", fmt.Errorf("persist exported artifact metadata: %w", err)
	}

	result := artifacts.SendFileResult{
		Message:  fmt.Sprintf("Sent file: %s", displayName),
		Artifact: artifacts.PublicSessionFileArtifactFromMetadata(artifact),
	}
	encoded, err := artifacts.EncodeSendFileResult(result)
	if err != nil {
		return "", err
	}
	return encoded, nil
}

func buildSendFileArtifactID(traceID string, toolCallID string) string {
	// Must remain compatible with artifacts.NormalizeArtifactID: [A-Za-z0-9_-] and length cap.
	stamp := time.Now().UTC().Format("20060102T150405_000000000Z")
	tracePart := sanitizeSendFileToken(traceID)
	callPart := sanitizeSendFileToken(toolCallID)
	if tracePart == "" {
		tracePart = "trace"
	}

	truncate := func(value string, maxLen int) string {
		if maxLen <= 0 {
			return ""
		}
		if len(value) <= maxLen {
			return value
		}
		return value[:maxLen]
	}

	maxLen := artifacts.MaxIdentifierLength
	if callPart == "" {
		tracePart = truncate(tracePart, maxLen-len(stamp)-1)
		if tracePart == "" {
			tracePart = "trace"
		}
		return fmt.Sprintf("%s-%s", stamp, tracePart)
	}

	overhead := len(stamp) + 2
	remaining := maxLen - overhead
	if remaining <= 0 {
		return truncate(stamp, maxLen)
	}

	callPart = truncate(callPart, remaining-len(tracePart))
	if callPart == "" {
		tracePart = truncate(tracePart, maxLen-len(stamp)-1)
		if tracePart == "" {
			tracePart = "trace"
		}
		return fmt.Sprintf("%s-%s", stamp, tracePart)
	}

	if len(tracePart)+len(callPart) > remaining {
		tracePart = truncate(tracePart, remaining-len(callPart))
	}
	if len(tracePart)+len(callPart) > remaining {
		callPart = truncate(callPart, remaining-len(tracePart))
	}
	if tracePart == "" {
		tracePart = "trace"
		callPart = truncate(callPart, remaining-len(tracePart))
	}

	return fmt.Sprintf("%s-%s-%s", stamp, tracePart, callPart)
}

func sanitizeSendFileToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			builder.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
		case ch == '-' || ch == '_':
			builder.WriteRune(ch)
		}
	}
	return strings.Trim(builder.String(), "-_")
}

func normalizeSendFileName(title string, filename string) string {
	title = strings.TrimSpace(title)
	filename = strings.TrimSpace(filename)
	if title == "" {
		return filename
	}
	ext := filepath.Ext(filename)
	if ext != "" && filepath.Ext(title) == "" {
		return title + ext
	}
	return title
}
