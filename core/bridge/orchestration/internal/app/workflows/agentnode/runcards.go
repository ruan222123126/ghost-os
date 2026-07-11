package agentnode

import (
	"context"
	"time"

	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type RunCardRecorder struct{}

func (RunCardRecorder) StartWorkflowAgentCard(
	ctx context.Context,
	req appworkflows.AgentRequest,
	sourceSessionID string,
) (RunCard, error) {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil, nil
	}
	handle, err := recorder.StartCard(ctx, runcards.StartInput{
		Kind:            bridgeTasks.RunCardKindWorkflowAgent,
		Title:           req.NodeID,
		NodeID:          req.NodeID,
		NodeType:        req.NodeType,
		BranchID:        req.BranchID,
		Iteration:       req.Iteration,
		SourceSessionID: sourceSessionID,
		StartedAt:       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	return runCard{handle: handle}, nil
}

type runCard struct {
	handle *runcards.Handle
}

func (c runCard) Sink() streaming.Sink {
	return runcards.NewStreamSink(c.handle)
}

func (c runCard) Finish(
	ctx context.Context,
	result bridgeTasks.ExecutionResult,
) error {
	return c.handle.Finish(ctx, runcards.FinishInput{
		Status:          result.Status,
		Preview:         result.ResponsePreview,
		ErrorText:       result.Error,
		SourceSessionID: result.SessionIDOutput,
		FinishedAt:      time.Now().UTC(),
	})
}
