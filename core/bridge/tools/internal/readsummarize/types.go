package readsummarize

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

const (
	ChunkLines            = 200
	DefaultMaxFiles       = 20
	DefaultMaxParallel    = 4
	DefaultMaxChunks      = 4
	MaxChunkBudget        = 8
	chunkSummaryMaxWord   = 180
	fileSummaryMaxWord    = 220
	crossFileSummaryWords = 260
)

type Worker interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type ExecutionClient interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

type Config struct {
	WorkerModel      string
	MaxFiles         int
	MaxParallel      int
	DefaultMaxChunks int
}

type FileReadChunk struct {
	ResolvedPath string
	StartLine    int
	EndLine      int
	TotalLines   int
	Content      string
}

type FileSummaryResult struct {
	RequestedPath string
	ResolvedPath  string
	Summary       string
	Error         string
	ChunksRead    int
	TotalLines    int
	LinesCovered  int
	Truncated     bool
}

type Service struct {
	runner    Runner
	formatter Formatter
}

func NormalizeConfig(cfg Config) Config {
	if cfg.MaxFiles <= 0 {
		cfg.MaxFiles = DefaultMaxFiles
	}
	if cfg.MaxParallel <= 0 {
		cfg.MaxParallel = DefaultMaxParallel
	}
	if cfg.DefaultMaxChunks <= 0 {
		cfg.DefaultMaxChunks = DefaultMaxChunks
	}
	if cfg.DefaultMaxChunks > MaxChunkBudget {
		cfg.DefaultMaxChunks = MaxChunkBudget
	}
	return cfg
}

func NormalizePaths(paths []string, maxFiles int) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("paths is required")
	}

	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
		if len(out) >= maxFiles {
			break
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("paths is required")
	}
	return out, nil
}

func ResolveChunkBudget(defaultBudget int, requested int) int {
	budget := defaultBudget
	if requested > 0 {
		budget = requested
	}
	if budget > MaxChunkBudget {
		budget = MaxChunkBudget
	}
	if budget <= 0 {
		budget = DefaultMaxChunks
	}
	return budget
}

func NewService(execution ExecutionClient, worker Worker, cfg Config) Service {
	cfg = NormalizeConfig(cfg)
	summarizer := NewWorkerClient(worker)
	return Service{
		runner: Runner{
			reader:      NewChunkReader(execution),
			summarizer:  summarizer,
			maxParallel: cfg.MaxParallel,
		},
		formatter: Formatter{
			workerModel: cfg.WorkerModel,
			summarizer:  summarizer,
		},
	}
}

func (s Service) Summarize(ctx context.Context, task string, paths []string, chunkBudget int, traceID string) (string, error) {
	results := s.runner.SummarizePaths(ctx, paths, task, chunkBudget, traceID)
	return s.formatter.Format(ctx, task, results)
}
