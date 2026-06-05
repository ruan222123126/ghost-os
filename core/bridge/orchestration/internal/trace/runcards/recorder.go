package runcards

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type Recorder struct {
	hub            *internaltrace.SessionPushHub
	sessionID      string
	runID          string
	progressWriter bridgeTasks.RunProgressWriter

	mu          sync.Mutex
	seq         int
	cards       []bridgeTasks.RunCard
	index       map[string]int
	cardStates  map[string]cardPersistState
	currentTime func() time.Time
}

type StartInput struct {
	Kind, Title      string
	NodeID, NodeType string
	BranchID         string
	SourceSessionID  string
	Round, Iteration int
	StartedAt        time.Time
}

type FinishInput struct {
	Status, Preview string
	ErrorText       string
	FinalText       string
	SourceSessionID string
	FinishedAt      time.Time
}

type Handle struct {
	recorder *Recorder
	cardID   string
}

func NewRecorder(
	ctx context.Context,
	hub *internaltrace.SessionPushHub,
) (*Recorder, error) {
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
	return &Recorder{
		hub:            hub,
		sessionID:      sessionID,
		runID:          strings.TrimSpace(runSession.RunID),
		progressWriter: runSession.ProgressWriter,
		cards:          make([]bridgeTasks.RunCard, 0, 8),
		index:          make(map[string]int),
		cardStates:     make(map[string]cardPersistState),
		currentTime:    time.Now,
	}, nil
}

func (h *Handle) Finish(ctx context.Context, input FinishInput) error {
	if h == nil || h.recorder == nil {
		return errors.New("task run card recorder is not configured")
	}
	return h.recorder.FinishCard(ctx, h.cardID, input)
}

func (r *Recorder) StartCard(ctx context.Context, input StartInput) (*Handle, error) {
	card, snapshot := r.appendCard(input)
	if err := r.publish(ctx, internaltrace.SessionPushTaskRunCardStarted, startedPayload(card)); err != nil {
		return nil, err
	}
	if err := r.persist(snapshot); err != nil {
		return nil, err
	}
	return &Handle{recorder: r, cardID: card.CardID}, nil
}

func (r *Recorder) appendCard(input StartInput) (bridgeTasks.RunCard, []bridgeTasks.RunCard) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	startedAt := input.StartedAt.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	card := bridgeTasks.RunCard{
		CardID:          fmt.Sprintf("%s-card-%03d", r.runID, r.seq),
		RunID:           r.runID,
		Kind:            strings.TrimSpace(input.Kind),
		Title:           strings.TrimSpace(input.Title),
		NodeID:          strings.TrimSpace(input.NodeID),
		NodeType:        strings.TrimSpace(input.NodeType),
		Round:           input.Round,
		Iteration:       input.Iteration,
		BranchID:        strings.TrimSpace(input.BranchID),
		SourceSessionID: strings.TrimSpace(input.SourceSessionID),
		StartedAt:       startedAt,
		Status:          bridgeTasks.RunStatusRunning,
	}
	r.index[card.CardID] = len(r.cards)
	r.cards = append(r.cards, card)
	r.cardStates[card.CardID] = cardPersistState{}
	return card, bridgeTasks.CloneRunCards(r.cards)
}

func (r *Recorder) FinishCard(ctx context.Context, cardID string, input FinishInput) error {
	card, snapshot, err := r.finishCard(cardID, input)
	if err != nil {
		return err
	}
	if err := r.publish(ctx, internaltrace.SessionPushTaskRunCardFinished, finishedPayload(card)); err != nil {
		return err
	}
	return r.persist(snapshot)
}

func (r *Recorder) finishCard(
	cardID string,
	input FinishInput,
) (bridgeTasks.RunCard, []bridgeTasks.RunCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	index, ok := r.index[strings.TrimSpace(cardID)]
	if !ok {
		return bridgeTasks.RunCard{}, nil, fmt.Errorf("task run card %q is not registered", cardID)
	}
	card := r.cards[index]
	card.Status = strings.TrimSpace(input.Status)
	card.Preview = strings.TrimSpace(input.Preview)
	card.Error = strings.TrimSpace(input.ErrorText)
	card.FinalText = strings.TrimSpace(input.FinalText)
	var assignErr error
	card, assignErr = assignSourceSessionID(card, input.SourceSessionID, "finish")
	if assignErr != nil {
		return bridgeTasks.RunCard{}, nil, assignErr
	}
	finishedAt := input.FinishedAt.UTC()
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	card.FinishedAt = finishedAt
	r.cards[index] = card
	return card, bridgeTasks.CloneRunCards(r.cards), nil
}

func (r *Recorder) EmitCardEvent(
	ctx context.Context,
	cardID string,
	event streaming.Event,
) error {
	snapshot, err := r.recordCardEvent(cardID, event)
	if err != nil {
		return err
	}
	if len(snapshot) > 0 {
		if err := r.persist(snapshot); err != nil {
			return err
		}
	}
	return r.publish(ctx, internaltrace.SessionPushTaskRunCardEvent, eventPayload(cardID, event))
}

func (r *Recorder) recordCardEvent(
	cardID string,
	event streaming.Event,
) ([]bridgeTasks.RunCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	trimmedCardID := strings.TrimSpace(cardID)
	index, ok := r.index[trimmedCardID]
	if !ok {
		return nil, fmt.Errorf("task run card %q is not registered", cardID)
	}

	card := r.cards[index]
	updated, err := assignSourceSessionID(card, event.SessionID, "event")
	if err != nil {
		return nil, err
	}
	now := r.nowUTC()
	updated.SourceEvents = appendRunCardSourceEvent(
		updated.SourceEvents,
		newRunCardSourceEvent(event, now),
	)
	r.cards[index] = updated
	state := r.cardStates[trimmedCardID]
	if !shouldPersistRunCardEvent(event.Type, state.lastPersistedAt, now) {
		return nil, nil
	}
	state.lastPersistedAt = now
	r.cardStates[trimmedCardID] = state
	return bridgeTasks.CloneRunCards(r.cards), nil
}

func (r *Recorder) Snapshot() []bridgeTasks.RunCard {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return bridgeTasks.CloneRunCards(r.cards)
}

func (r *Recorder) persist(snapshot []bridgeTasks.RunCard) error {
	if r == nil || r.progressWriter == nil {
		return nil
	}
	return r.progressWriter.WriteRunningRunLog(bridgeTasks.RunningRunLogUpdate{
		RunCards: snapshot,
	})
}

func assignSourceSessionID(
	card bridgeTasks.RunCard,
	sourceSessionID string,
	source string,
) (bridgeTasks.RunCard, error) {
	next := strings.TrimSpace(sourceSessionID)
	if next == "" {
		return card, nil
	}
	current := strings.TrimSpace(card.SourceSessionID)
	if current != "" && current != next {
		return bridgeTasks.RunCard{}, fmt.Errorf(
			"task run card %q source_session_id mismatch: current=%q %s=%q",
			card.CardID,
			current,
			source,
			next,
		)
	}
	card.SourceSessionID = next
	return card, nil
}

func (r *Recorder) publish(
	_ context.Context,
	eventType internaltrace.SessionPushEventType,
	payload any,
) error {
	if r == nil || r.hub == nil {
		return errors.New("task run session push hub is not configured")
	}
	if r.sessionID == "" {
		return errors.New("task run display session is not configured")
	}
	r.hub.Publish(internaltrace.SessionPushEvent{
		Type:      eventType,
		SessionID: r.sessionID,
		Payload:   payload,
		At:        time.Now().UTC(),
	})
	return nil
}

func (r *Recorder) nowUTC() time.Time {
	if r == nil || r.currentTime == nil {
		return time.Now().UTC()
	}
	return r.currentTime().UTC()
}
