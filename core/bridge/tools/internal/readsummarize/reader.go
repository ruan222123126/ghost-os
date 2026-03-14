package readsummarize

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type ChunkReader struct {
	execution ExecutionClient
}

func NewChunkReader(execution ExecutionClient) ChunkReader {
	return ChunkReader{execution: execution}
}

func (r ChunkReader) Read(
	ctx context.Context,
	path string,
	chunkBudget int,
	traceID string,
) ([]FileReadChunk, bool, error) {
	chunks := make([]FileReadChunk, 0, chunkBudget)
	startLine := 1
	truncated := false

	for chunkIndex := 0; chunkIndex < chunkBudget; chunkIndex++ {
		endLine := startLine + ChunkLines - 1
		payload, err := r.execution.Call(ctx, "READ_FILE", map[string]any{
			"path":       path,
			"start_line": startLine,
			"end_line":   endLine,
		}, traceID)
		if err != nil {
			return nil, false, fmt.Errorf("execution READ_FILE failed for %q: %w", path, err)
		}

		chunk, err := decodeChunk(payload)
		if err != nil {
			return nil, false, fmt.Errorf("invalid READ_FILE payload for %q: %w", path, err)
		}
		if chunk.TotalLines == 0 || chunk.StartLine == 0 || chunk.EndLine == 0 {
			if len(chunks) == 0 {
				chunk.ResolvedPath = strings.TrimSpace(chunk.ResolvedPath)
				if chunk.ResolvedPath == "" {
					chunk.ResolvedPath = strings.TrimSpace(path)
				}
				chunks = append(chunks, chunk)
			}
			return chunks, false, nil
		}

		chunks = append(chunks, chunk)
		if chunk.EndLine >= chunk.TotalLines {
			return chunks, false, nil
		}
		startLine = chunk.EndLine + 1
	}

	if len(chunks) > 0 && chunks[len(chunks)-1].EndLine < chunks[len(chunks)-1].TotalLines {
		truncated = true
	}
	return chunks, truncated, nil
}

func decodeChunk(payload map[string]any) (FileReadChunk, error) {
	resolvedPath, err := payloadutil.String(payload, "path")
	if err != nil {
		return FileReadChunk{}, err
	}
	returnedStartLine, err := payloadutil.Int(payload, "returned_start_line")
	if err != nil {
		return FileReadChunk{}, err
	}
	returnedEndLine, err := payloadutil.Int(payload, "returned_end_line")
	if err != nil {
		return FileReadChunk{}, err
	}
	totalLines, err := payloadutil.Int(payload, "total_lines")
	if err != nil {
		return FileReadChunk{}, err
	}
	content, err := payloadutil.String(payload, "content")
	if err != nil {
		return FileReadChunk{}, err
	}
	return FileReadChunk{
		ResolvedPath: strings.TrimSpace(resolvedPath),
		StartLine:    returnedStartLine,
		EndLine:      returnedEndLine,
		TotalLines:   totalLines,
		Content:      content,
	}, nil
}
