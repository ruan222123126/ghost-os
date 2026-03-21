package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools/internal/readsummarize"
)

type readAndSummarizeWorker interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type ReadAndSummarizeConfig struct {
	WorkerModel      string
	MaxFiles         int
	MaxParallel      int
	DefaultMaxChunks int
}

type ReadAndSummarizeTool struct {
	maxFiles         int
	defaultMaxChunks int
	service          readsummarize.Service
	setupErr         error
}

type readAndSummarizeArgs struct {
	Paths            []string `json:"paths"`
	Task             string   `json:"task"`
	MaxChunksPerFile int      `json:"max_chunks_per_file,omitempty"`
}

func NewReadAndSummarizeTool(client ExecutionClient, worker readAndSummarizeWorker, cfg ReadAndSummarizeConfig) Tool {
	internalCfg := readsummarize.NormalizeConfig(readsummarize.Config{
		WorkerModel:      cfg.WorkerModel,
		MaxFiles:         cfg.MaxFiles,
		MaxParallel:      cfg.MaxParallel,
		DefaultMaxChunks: cfg.DefaultMaxChunks,
	})

	var setupErr error
	switch {
	case client == nil:
		setupErr = fmt.Errorf("execution client is not configured")
	case worker == nil:
		setupErr = fmt.Errorf("worker model is not configured")
	}

	return ReadAndSummarizeTool{
		maxFiles:         internalCfg.MaxFiles,
		defaultMaxChunks: internalCfg.DefaultMaxChunks,
		service:          readsummarize.NewService(client, worker, internalCfg),
		setupErr:         setupErr,
	}
}

func (ReadAndSummarizeTool) Name() string {
	return "read_and_summarize"
}

func (ReadAndSummarizeTool) Description() string {
	return "Read multiple local files and summarize them with a worker model for fast triage. Use the available workspace tools to verify exact code before editing."
}

func (ReadAndSummarizeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"paths":{
				"type":"array",
				"items":{"type":"string"},
				"minItems":1,
				"maxItems":20,
				"description":"File paths to summarize for broad codebase triage."
			},
			"task":{
				"type":"string",
				"description":"Specific question or focus for the summaries, e.g. 'Find auth flow ownership' or 'Locate where session end is emitted'."
			},
			"max_chunks_per_file":{
				"type":"integer",
				"minimum":1,
				"maximum":8,
				"description":"Optional per-file read budget in 200-line chunks. Defaults to the worker budget configured by the bridge."
			}
		},
		"required":["paths","task"],
		"additionalProperties":false
	}`)
}

func (t ReadAndSummarizeTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.setupErr != nil {
		return "", t.setupErr
	}

	var args readAndSummarizeArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	task := strings.TrimSpace(args.Task)
	if task == "" {
		return "", fmt.Errorf("task is required")
	}

	paths, err := readsummarize.NormalizePaths(args.Paths, t.maxFiles)
	if err != nil {
		return "", err
	}
	chunkBudget := readsummarize.ResolveChunkBudget(t.defaultMaxChunks, args.MaxChunksPerFile)
	return t.service.Summarize(ctx, task, paths, chunkBudget, traceID)
}
