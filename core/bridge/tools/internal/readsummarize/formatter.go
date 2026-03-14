package readsummarize

import (
	"context"
	"fmt"
	"strings"
)

type Formatter struct {
	workerModel string
	summarizer  WorkerClient
}

func (f Formatter) Format(ctx context.Context, task string, results []FileSummaryResult) (string, error) {
	var builder strings.Builder
	builder.WriteString("Read & Summarize\n")
	builder.WriteString(fmt.Sprintf("Focus: %s\n", task))
	if f.workerModel != "" {
		builder.WriteString(fmt.Sprintf("Worker model: %s\n", f.workerModel))
	}
	builder.WriteString(fmt.Sprintf("Files requested: %d\n", len(results)))

	successCount := 0
	for _, result := range results {
		builder.WriteByte('\n')
		path := result.ResolvedPath
		if path == "" {
			path = result.RequestedPath
		}
		builder.WriteString(fmt.Sprintf("File: %s\n", path))
		if strings.TrimSpace(result.Error) != "" {
			builder.WriteString(fmt.Sprintf("Status: error\nError: %s\n", result.Error))
			continue
		}
		successCount++
		status := "complete"
		if result.Truncated {
			status = "partial"
		}
		builder.WriteString(fmt.Sprintf("Status: %s\n", status))
		if result.TotalLines > 0 {
			builder.WriteString(fmt.Sprintf("Coverage: %d/%d lines across %d chunk(s)\n", result.LinesCovered, result.TotalLines, result.ChunksRead))
		}
		builder.WriteString("Summary:\n")
		builder.WriteString(strings.TrimSpace(result.Summary))
		builder.WriteByte('\n')
	}

	if successCount == 0 {
		return strings.TrimSpace(builder.String()), nil
	}

	crossFile, err := f.summarizer.SummarizeCrossFileSet(ctx, task, results)
	if err == nil && strings.TrimSpace(crossFile) != "" {
		builder.WriteString("\nCross-file synthesis:\n")
		builder.WriteString(strings.TrimSpace(crossFile))
		builder.WriteByte('\n')
	}

	return strings.TrimSpace(builder.String()), nil
}
