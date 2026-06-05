package runcards

import (
	"encoding/json"
	"strings"
	"time"

	apicontracts "ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

func startedPayload(card bridgeTasks.RunCard) apicontracts.TaskRunCardStartedPayload {
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

func eventPayload(cardID string, event streaming.Event) apicontracts.TaskRunCardEventPayload {
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
			Payload:   payloadRecord(event.Payload),
			At:        event.At.Format(time.RFC3339Nano),
		},
	}
}

func finishedPayload(card bridgeTasks.RunCard) apicontracts.TaskRunCardFinishedPayload {
	return apicontracts.TaskRunCardFinishedPayload{
		CardID:          card.CardID,
		Status:          card.Status,
		FinishedAt:      card.FinishedAt.Format(time.RFC3339Nano),
		Preview:         card.Preview,
		Error:           card.Error,
		SourceSessionID: card.SourceSessionID,
	}
}

func payloadRecord(payload any) map[string]any {
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
