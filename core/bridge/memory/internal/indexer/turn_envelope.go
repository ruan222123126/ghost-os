package indexer

import (
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type turnState struct {
	traceID      string
	messages     map[int]TurnMessage
	orderedIndex []int
	startIndex   int
	count        int
	startedAt    time.Time
	committedAt  time.Time
	commitOffset int64
	commitEvent  string
	commitIndex  int
	lastEventID  string
}

func BuildTurnEnvelopes(bucket Bucket, events []Event) []TurnEnvelope {
	if len(events) == 0 {
		return nil
	}
	sorted := append([]Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Offset == sorted[j].Offset {
			if sorted[i].OccurredAt.Equal(sorted[j].OccurredAt) {
				return sorted[i].EventID < sorted[j].EventID
			}
			return sorted[i].OccurredAt.Before(sorted[j].OccurredAt)
		}
		return sorted[i].Offset < sorted[j].Offset
	})
	states := make(map[string]*turnState, len(sorted))
	completed := make([]TurnEnvelope, 0, len(sorted)/2)
	for _, event := range sorted {
		key := envelopeTurnKey(event)
		state, ok := states[key]
		if !ok {
			state = &turnState{messages: make(map[int]TurnMessage, 4), startIndex: -1, commitIndex: -1}
			states[key] = state
		}
		if state.traceID == "" {
			state.traceID = strings.TrimSpace(event.TraceID)
		}
		if state.startedAt.IsZero() || (!event.OccurredAt.IsZero() && event.OccurredAt.Before(state.startedAt)) {
			state.startedAt = event.OccurredAt.UTC()
		}
		state.lastEventID = strings.TrimSpace(event.EventID)
		switch strings.TrimSpace(event.Kind) {
		case "message_appended":
			if event.Payload.Message == nil {
				continue
			}
			if _, exists := state.messages[event.MessageIndex]; !exists {
				cloned := llm.CloneMessages([]llm.Message{*event.Payload.Message})
				if len(cloned) == 0 {
					continue
				}
				state.messages[event.MessageIndex] = TurnMessage{Index: event.MessageIndex, EventID: strings.TrimSpace(event.EventID), Offset: event.Offset, Message: cloned[0]}
				state.orderedIndex = append(state.orderedIndex, event.MessageIndex)
				if state.startIndex < 0 || event.MessageIndex < state.startIndex {
					state.startIndex = event.MessageIndex
				}
			}
		case "turn_committed":
			if event.Payload.StartIndex >= 0 {
				state.startIndex = event.Payload.StartIndex
			}
			state.count = event.Payload.MessageCount
			state.committedAt = event.OccurredAt.UTC()
			state.commitOffset = event.Offset
			state.commitEvent = strings.TrimSpace(event.EventID)
			state.commitIndex++
			envelope := buildEnvelope(bucket, event, state)
			if envelope.CommitEventID != "" {
				completed = append(completed, envelope)
			}
		}
	}
	sort.SliceStable(completed, func(i, j int) bool {
		if completed[i].CommitOffset == completed[j].CommitOffset {
			return completed[i].CommitEventID < completed[j].CommitEventID
		}
		return completed[i].CommitOffset < completed[j].CommitOffset
	})
	return completed
}

func buildEnvelope(bucket Bucket, commit Event, state *turnState) TurnEnvelope {
	ordered := append([]int(nil), state.orderedIndex...)
	sort.Ints(ordered)
	messages := make([]llm.Message, 0, len(ordered))
	messageEvents := make([]TurnMessage, 0, len(ordered))
	for _, index := range ordered {
		messageEvent, ok := state.messages[index]
		if !ok {
			continue
		}
		messages = append(messages, messageEvent.Message)
		messageEvents = append(messageEvents, messageEvent)
	}
	messageCount := state.count
	if messageCount <= 0 {
		messageCount = len(messages)
	}
	startIndex := state.startIndex
	if startIndex < 0 && len(messageEvents) > 0 {
		startIndex = messageEvents[0].Index
	}
	startedAt := state.startedAt
	if startedAt.IsZero() {
		startedAt = commit.OccurredAt.UTC()
	}
	return TurnEnvelope{
		Bucket:         normalizeBucket(bucket),
		Namespace:      firstNonEmpty(strings.TrimSpace(commit.Namespace), normalizeBucket(bucket).Namespace),
		Workspace:      firstNonEmpty(strings.TrimSpace(commit.Workspace), normalizeBucket(bucket).Workspace),
		SessionID:      strings.TrimSpace(commit.SessionID),
		TurnID:         strings.TrimSpace(commit.TurnID),
		TraceID:        firstNonEmpty(state.traceID, strings.TrimSpace(commit.TraceID)),
		StartIndex:     startIndex,
		MessageCount:   messageCount,
		Messages:       llm.CloneMessages(messages),
		MessageEvents:  append([]TurnMessage(nil), messageEvents...),
		CommitEventID:  state.commitEvent,
		CommitOffset:   state.commitOffset,
		LastEventID:    firstNonEmpty(state.lastEventID, state.commitEvent),
		StartedAt:      startedAt,
		CommittedAt:    state.committedAt,
		CommittedIndex: state.commitIndex,
	}
}

func envelopeTurnKey(event Event) string {
	turnID := strings.TrimSpace(event.TurnID)
	if turnID == "" {
		turnID = firstNonEmpty(strings.TrimSpace(event.TraceID), strings.TrimSpace(event.EventID))
	}
	return strings.TrimSpace(event.SessionID) + "|" + turnID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
