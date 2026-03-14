package readsummarize

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

type WorkerClient struct {
	worker Worker
}

func NewWorkerClient(worker Worker) WorkerClient {
	return WorkerClient{worker: worker}
}

func (c WorkerClient) SummarizeWholeFile(ctx context.Context, task string, chunk FileReadChunk) (string, error) {
	user := fmt.Sprintf(
		"Task:\n%s\n\nFile: %s\nLines: %d-%d of %d\n\nContent:\n%s\n\nReturn a concise implementation-focused summary for the orchestrator. Mention the main symbols, control flow, and anything relevant to the task. Keep it under %d words.",
		task,
		chunk.ResolvedPath,
		chunk.StartLine,
		chunk.EndLine,
		chunk.TotalLines,
		chunk.Content,
		fileSummaryMaxWord,
	)
	return c.complete(ctx, workerSystemPrompt(), user)
}

func (c WorkerClient) SummarizeChunk(ctx context.Context, task string, chunk FileReadChunk) (string, error) {
	user := fmt.Sprintf(
		"Task:\n%s\n\nFile: %s\nChunk lines: %d-%d of %d\n\nContent:\n%s\n\nSummarize only this chunk for the orchestrator. Call out symbols, decisions, TODOs, and anything relevant to the task. Keep it under %d words.",
		task,
		chunk.ResolvedPath,
		chunk.StartLine,
		chunk.EndLine,
		chunk.TotalLines,
		chunk.Content,
		chunkSummaryMaxWord,
	)
	return c.complete(ctx, workerSystemPrompt(), user)
}

func (c WorkerClient) SummarizeChunkSet(
	ctx context.Context,
	task string,
	path string,
	chunkSummaries []string,
	totalLines int,
	truncated bool,
) (string, error) {
	joined := make([]string, 0, len(chunkSummaries))
	for i, summary := range chunkSummaries {
		joined = append(joined, fmt.Sprintf("Chunk %d:\n%s", i+1, strings.TrimSpace(summary)))
	}
	status := "complete"
	if truncated {
		status = "partial"
	}
	user := fmt.Sprintf(
		"Task:\n%s\n\nFile: %s\nCoverage: %s read of %d lines\n\nChunk summaries:\n%s\n\nMerge these chunk summaries into one file-level summary for the orchestrator. Highlight the main responsibilities, relevant symbols, and any uncertainty caused by partial coverage. Keep it under %d words.",
		task,
		path,
		status,
		totalLines,
		strings.Join(joined, "\n\n"),
		fileSummaryMaxWord,
	)
	return c.complete(ctx, workerSystemPrompt(), user)
}

func (c WorkerClient) SummarizeCrossFileSet(ctx context.Context, task string, results []FileSummaryResult) (string, error) {
	parts := make([]string, 0, len(results))
	for _, result := range results {
		if strings.TrimSpace(result.Error) != "" || strings.TrimSpace(result.Summary) == "" {
			continue
		}
		path := result.ResolvedPath
		if path == "" {
			path = result.RequestedPath
		}
		coverage := fmt.Sprintf("%d/%d lines", result.LinesCovered, result.TotalLines)
		if result.TotalLines == 0 {
			coverage = "empty file"
		}
		status := "complete"
		if result.Truncated {
			status = "partial"
		}
		parts = append(parts, fmt.Sprintf("File: %s\nCoverage: %s (%s)\nSummary:\n%s", path, coverage, status, strings.TrimSpace(result.Summary)))
	}
	if len(parts) <= 1 {
		return "", nil
	}

	user := fmt.Sprintf(
		"Task:\n%s\n\nFile summaries:\n%s\n\nProduce a cross-file synthesis for the orchestrator. Explain how these files relate, where the likely ownership sits, and what should be inspected next. Keep it under %d words.",
		task,
		strings.Join(parts, "\n\n"),
		crossFileSummaryWords,
	)
	return c.complete(ctx, workerSystemPrompt(), user)
}

func workerSystemPrompt() string {
	return "You are a Ghost-OS worker model specialized in fast code triage. Summarize code accurately for a stronger orchestrator model. Be concise, factual, and implementation-focused. Do not invent behavior that is not visible in the provided text."
}

func (c WorkerClient) complete(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	resp, err := c.worker.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: systemPrompt},
			{Role: llm.RoleUser, Text: userPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("worker summarize failed: %w", err)
	}
	summary := strings.TrimSpace(resp.Message.Text)
	if summary == "" {
		return "", fmt.Errorf("worker summarize returned empty content")
	}
	return summary, nil
}
