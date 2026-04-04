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

type sendFileExportResult struct {
	Filename     string
	StoredPath   string
	OriginalPath string
	MIMEType     string
	SHA256       string
	Bytes        int
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
	args, path, err := decodeSendFileArgs(argsJSON)
	if err != nil {
		return "", err
	}
	sessionID, err := resolveSendFileSessionID(ctx)
	if err != nil {
		return "", err
	}
	artifactID := buildSendFileArtifactID(traceID, ToolCallIDFromContext(ctx))
	payload, err := t.execution.Call(ctx, "EXPORT_FILE", map[string]any{
		"path":          path,
		"session_id":    sessionID,
		"artifact_root": t.store.BaseDir(),
		"artifact_id":   artifactID,
		"max_bytes":     defaultSendFileMaxBytes,
	}, traceID)
	if err != nil {
		return "", fmt.Errorf("execution EXPORT_FILE failed: %w", err)
	}
	exported, err := decodeSendFileExportResult(payload)
	if err != nil {
		return "", err
	}

	displayName := normalizeSendFileName(strings.TrimSpace(args.Title), exported.Filename)
	artifact := artifacts.SessionFileArtifact{
		ArtifactID:  artifactID,
		SessionID:   sessionID,
		Name:        displayName,
		MimeType:    strings.TrimSpace(exported.MIMEType),
		Bytes:       int64(exported.Bytes),
		SHA256:      strings.TrimSpace(exported.SHA256),
		DownloadURL: fmt.Sprintf("/api/sessions/%s/artifacts/%s", sessionID, artifactID),
		SourcePath:  strings.TrimSpace(exported.OriginalPath),
		StoredPath:  strings.TrimSpace(exported.StoredPath),
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

func decodeSendFileArgs(argsJSON json.RawMessage) (sendFileArgs, string, error) {
	var args sendFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return sendFileArgs{}, "", fmt.Errorf("decode args: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return sendFileArgs{}, "", fmt.Errorf("path is required")
	}
	return args, path, nil
}

func resolveSendFileSessionID(ctx context.Context) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("send_file requires an active session")
	}
	sessionID := strings.TrimSpace(sess.ID)
	if sessionID == "" {
		return "", fmt.Errorf("send_file requires an active session")
	}
	return sessionID, nil
}

func decodeSendFileExportResult(payload map[string]any) (sendFileExportResult, error) {
	filename, err := payloadutil.String(payload, "filename")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	storedPath, err := payloadutil.String(payload, "stored_path")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	originalPath, err := payloadutil.String(payload, "original_path")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	mimeType, err := payloadutil.String(payload, "mime_type")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	sha256, err := payloadutil.String(payload, "sha256")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	bytesCount, err := payloadutil.Int(payload, "bytes")
	if err != nil {
		return sendFileExportResult{}, fmt.Errorf("invalid EXPORT_FILE payload: %w", err)
	}
	return sendFileExportResult{
		Filename:     filename,
		StoredPath:   storedPath,
		OriginalPath: originalPath,
		MIMEType:     mimeType,
		SHA256:       sha256,
		Bytes:        bytesCount,
	}, nil
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
