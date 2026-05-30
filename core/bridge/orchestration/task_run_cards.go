package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	apicontracts "ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type taskRunCardRecorder struct {
	hub            *sessionPushHub
	sessionID      string
	runID          string
	progressWriter bridgeTasks.RunProgressWriter

	mu    sync.Mutex
	seq   int
	cards []bridgeTasks.RunCard
	index map[string]int
}

type taskRunCardStartInput struct {
	kind            string
	title           string
	nodeID          string
	nodeType        string
	round           int
	iteration       int
	branchID        string
	sourceSessionID string
	startedAt       time.Time
}

type taskRunCardFinishInput struct {
	status          string
	preview         string
	errorText       string
	finalText       string
	sourceSessionID string
	finishedAt      time.Time
}

type taskRunCardHandle struct {
	recorder *taskRunCardRecorder
	cardID   string
}

func (h *taskRunCardHandle) Finish(
	ctx context.Context,
	input taskRunCardFinishInput,
) error {
	if h == nil || h.recorder == nil {
		return errors.New("task run card recorder is not configured")
	}
	return h.recorder.FinishCard(ctx, h.cardID, input)
}

func newTaskRunCardRecorder(
	ctx context.Context,
	hub *sessionPushHub,
) (*taskRunCardRecorder, error) {
	runSession, ok := bridgeTasks.RunSessionFromContext(ctx)
	if !ok {
		return nil, errors.New("task run session is not configured")
	}
	sessionID := strings.TrimSpace(runSession.SessionID)
	if sessionID == "" {
		return nil, errors.New("task run display session is not configured")
	}
	if strings.TrimSpace(runSession.RunID) == "" {
		return nil, errors.New("task run id is not configured")
	}
	if hub == nil {
		return nil, errors.New("task run session push hub is not configured")
	}
	return &taskRunCardRecorder{
		hub:            hub,
		sessionID:      sessionID,
		runID:          strings.TrimSpace(runSession.RunID),
		progressWriter: runSession.ProgressWriter,
		cards:          make([]bridgeTasks.RunCard, 0, 8),
		index:          make(map[string]int),
	}, nil
}

func (r *taskRunCardRecorder) StartCard(
	ctx context.Context,
	input taskRunCardStartInput,
) (*taskRunCardHandle, error) {
	card, snapshot := r.appendCard(input)
	if err := r.publish(ctx, sessionPushTaskRunCardStarted, buildTaskRunCardStartedPayload(card)); err != nil {
		return nil, err
	}
	if err := r.persist(snapshot); err != nil {
		return nil, err
	}
	return &taskRunCardHandle{recorder: r, cardID: card.CardID}, nil
}

func (r *taskRunCardRecorder) appendCard(
	input taskRunCardStartInput,
) (bridgeTasks.RunCard, []bridgeTasks.RunCard) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	startedAt := input.startedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	card := bridgeTasks.RunCard{
		CardID:          fmt.Sprintf("%s-card-%03d", r.runID, r.seq),
		RunID:           r.runID,
		Kind:            strings.TrimSpace(input.kind),
		Title:           strings.TrimSpace(input.title),
		NodeID:          strings.TrimSpace(input.nodeID),
		NodeType:        strings.TrimSpace(input.nodeType),
		Round:           input.round,
		Iteration:       input.iteration,
		BranchID:        strings.TrimSpace(input.branchID),
		SourceSessionID: strings.TrimSpace(input.sourceSessionID),
		StartedAt:       startedAt,
		Status:          taskRunStatusRunning,
	}
	r.index[card.CardID] = len(r.cards)
	r.cards = append(r.cards, card)
	return card, bridgeTasks.CloneRunCards(r.cards)
}

func (r *taskRunCardRecorder) ForwardEvent(
	ctx context.Context,
	cardID string,
	event streaming.Event,
) error {
	return r.publish(ctx, sessionPushTaskRunCardEvent, buildTaskRunCardEventPayload(cardID, event))
}

func (r *taskRunCardRecorder) FinishCard(
	ctx context.Context,
	cardID string,
	input taskRunCardFinishInput,
) error {
	card, snapshot, err := r.finishCard(cardID, input)
	if err != nil {
		return err
	}
	if err := r.publish(ctx, sessionPushTaskRunCardFinished, buildTaskRunCardFinishedPayload(card)); err != nil {
		return err
	}
	return r.persist(snapshot)
}

func (r *taskRunCardRecorder) finishCard(
	cardID string,
	input taskRunCardFinishInput,
) (bridgeTasks.RunCard, []bridgeTasks.RunCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	index, ok := r.index[strings.TrimSpace(cardID)]
	if !ok {
		return bridgeTasks.RunCard{}, nil, fmt.Errorf("task run card %q is not registered", cardID)
	}
	card := r.cards[index]
	card.Status = strings.TrimSpace(input.status)
	card.Preview = strings.TrimSpace(input.preview)
	card.Error = strings.TrimSpace(input.errorText)
	card.FinalText = strings.TrimSpace(input.finalText)
	if sourceSessionID := strings.TrimSpace(input.sourceSessionID); sourceSessionID != "" {
		card.SourceSessionID = sourceSessionID
	}
	finishedAt := input.finishedAt.UTC()
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	card.FinishedAt = finishedAt
	r.cards[index] = card
	return card, bridgeTasks.CloneRunCards(r.cards), nil
}

func (r *taskRunCardRecorder) Snapshot() []bridgeTasks.RunCard {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return bridgeTasks.CloneRunCards(r.cards)
}

func (r *taskRunCardRecorder) persist(snapshot []bridgeTasks.RunCard) error {
	if r == nil || r.progressWriter == nil {
		return nil
	}
	return r.progressWriter.WriteRunningRunLog(bridgeTasks.RunningRunLogUpdate{
		RunCards: snapshot,
	})
}

func (r *taskRunCardRecorder) publish(
	_ context.Context,
	eventType sessionPushEventType,
	payload any,
) error {
	if r == nil || r.hub == nil {
		return errors.New("task run session push hub is not configured")
	}
	if r.sessionID == "" {
		return errors.New("task run display session is not configured")
	}
	r.hub.Publish(sessionPushEvent{
		Type:      eventType,
		SessionID: r.sessionID,
		Payload:   payload,
		At:        time.Now().UTC(),
	})
	return nil
}

func buildTaskRunCardStartedPayload(
	card bridgeTasks.RunCard,
) apicontracts.TaskRunCardStartedPayload {
	return apicontracts.TaskRunCardStartedPayload{
		CardID:          card.CardID,
		RunID:           card.RunID,
		Kind:            card.Kind,
		Title:           card.Title,
		NodeID:          card.NodeID,
		NodeType:        card.NodeType,
		Round:           card.Round,
		Iteration:       card.Iteration,
		BranchID:        card.BranchID,
		SourceSessionID: card.SourceSessionID,
		StartedAt:       card.StartedAt.Format(time.RFC3339Nano),
	}
}

func buildTaskRunCardEventPayload(
	cardID string,
	event streaming.Event,
) apicontracts.TaskRunCardEventPayload {
	return apicontracts.TaskRunCardEventPayload{
		CardID:          strings.TrimSpace(cardID),
		SourceSessionID: strings.TrimSpace(event.SessionID),
		SourceEvent: apicontracts.AgentStreamEventContract{
			ID:        strings.TrimSpace(event.ID),
			StepID:    strings.TrimSpace(event.StepID),
			TraceID:   strings.TrimSpace(event.TraceID),
			SessionID: strings.TrimSpace(event.SessionID),
			Turn:      event.Turn,
			Type:      string(event.Type),
			Payload:   taskRunCardPayloadRecord(event.Payload),
			At:        event.At.Format(time.RFC3339Nano),
		},
	}
}

func buildTaskRunCardFinishedPayload(
	card bridgeTasks.RunCard,
) apicontracts.TaskRunCardFinishedPayload {
	return apicontracts.TaskRunCardFinishedPayload{
		CardID:          card.CardID,
		Status:          card.Status,
		FinishedAt:      card.FinishedAt.Format(time.RFC3339Nano),
		Preview:         card.Preview,
		Error:           card.Error,
		SourceSessionID: card.SourceSessionID,
	}
}

func taskRunCardPayloadRecord(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if record, ok := payload.(map[string]any); ok {
		return record
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{}
	}
	record := map[string]any{}
	if err := json.Unmarshal(encoded, &record); err != nil {
		return map[string]any{}
	}
	return record
}

type taskRunCardStreamSink struct {
	handle *taskRunCardHandle
}

func newTaskRunCardStreamSink(handle *taskRunCardHandle) streaming.Sink {
	return taskRunCardStreamSink{handle: handle}
}

func (s taskRunCardStreamSink) Emit(
	ctx context.Context,
	event streaming.Event,
) (streaming.Event, error) {
	if s.handle == nil || s.handle.recorder == nil {
		return event, nil
	}
	return event, s.handle.recorder.ForwardEvent(ctx, s.handle.cardID, event)
}
