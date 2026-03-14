package readsummarize

import (
	"context"
	"strings"
	"sync"
)

type Runner struct {
	reader      ChunkReader
	summarizer  WorkerClient
	maxParallel int
}

func NewRunner(reader ChunkReader, summarizer WorkerClient, maxParallel int) Runner {
	if maxParallel <= 0 {
		maxParallel = DefaultMaxParallel
	}
	return Runner{reader: reader, summarizer: summarizer, maxParallel: maxParallel}
}

func (r Runner) SummarizePaths(
	ctx context.Context,
	paths []string,
	task string,
	chunkBudget int,
	traceID string,
) []FileSummaryResult {
	results := make([]FileSummaryResult, len(paths))
	sem := make(chan struct{}, r.maxParallel)
	var wg sync.WaitGroup

	for i, path := range paths {
		wg.Add(1)
		go func(index int, requestedPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[index] = r.SummarizeFile(ctx, requestedPath, task, chunkBudget, traceID)
		}(i, path)
	}

	wg.Wait()
	return results
}

func (r Runner) SummarizeFile(
	ctx context.Context,
	path string,
	task string,
	chunkBudget int,
	traceID string,
) FileSummaryResult {
	chunks, truncated, err := r.reader.Read(ctx, path, chunkBudget, traceID)
	if err != nil {
		return FileSummaryResult{RequestedPath: path, Error: err.Error()}
	}
	if len(chunks) == 0 {
		return FileSummaryResult{
			RequestedPath: path,
			ResolvedPath:  path,
			Summary:       "File is empty.",
		}
	}
	if chunks[0].TotalLines == 0 {
		resolvedPath := strings.TrimSpace(chunks[0].ResolvedPath)
		if resolvedPath == "" {
			resolvedPath = path
		}
		return FileSummaryResult{
			RequestedPath: path,
			ResolvedPath:  resolvedPath,
			Summary:       "File is empty.",
		}
	}

	result := FileSummaryResult{
		RequestedPath: path,
		ResolvedPath:  chunks[0].ResolvedPath,
		ChunksRead:    len(chunks),
		TotalLines:    chunks[0].TotalLines,
		LinesCovered:  chunks[len(chunks)-1].EndLine,
		Truncated:     truncated,
	}

	if len(chunks) == 1 {
		summary, err := r.summarizer.SummarizeWholeFile(ctx, task, chunks[0])
		if err != nil {
			result.Error = err.Error()
			return result
		}
		result.Summary = summary
		return result
	}

	chunkSummaries, err := r.summarizeChunks(ctx, task, chunks)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	summary, err := r.summarizer.SummarizeChunkSet(ctx, task, chunks[0].ResolvedPath, chunkSummaries, result.TotalLines, result.Truncated)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Summary = summary
	return result
}

func (r Runner) summarizeChunks(ctx context.Context, task string, chunks []FileReadChunk) ([]string, error) {
	summaries := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		summary, err := r.summarizer.SummarizeChunk(ctx, task, chunk)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}
