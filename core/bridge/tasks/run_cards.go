package tasks

import (
	"strings"
	"time"
)

const (
	RunCardKindAgentTask           = "agent_task"
	RunCardKindWorkflowAgent       = "workflow_agent"
	RunCardKindWorkflowLLM         = "workflow_llm"
	RunCardKindOrchestrationOwner  = "orchestration_owner"
	RunCardKindOrchestrationMember = "orchestration_member"
	RunCardKindRelayRound          = "relay_round"
)

type RunCard struct {
	CardID          string               `json:"card_id"`
	RunID           string               `json:"run_id,omitempty"`
	Kind            string               `json:"kind"`
	Title           string               `json:"title,omitempty"`
	NodeID          string               `json:"node_id,omitempty"`
	NodeType        string               `json:"node_type,omitempty"`
	Round           int                  `json:"round,omitempty"`
	Iteration       int                  `json:"iteration,omitempty"`
	BranchID        string               `json:"branch_id,omitempty"`
	SourceSessionID string               `json:"source_session_id,omitempty"`
	StartedAt       time.Time            `json:"started_at,omitempty"`
	Status          string               `json:"status,omitempty"`
	FinishedAt      time.Time            `json:"finished_at,omitempty"`
	Preview         string               `json:"preview,omitempty"`
	Error           string               `json:"error,omitempty"`
	FinalText       string               `json:"final_text,omitempty"`
	SourceEvents    []RunCardSourceEvent `json:"source_events,omitempty"`
}

func CloneRunCards(input []RunCard) []RunCard {
	if len(input) == 0 {
		return nil
	}
	out := make([]RunCard, len(input))
	for index, item := range input {
		out[index] = normalizeRunCard(item)
	}
	return out
}

func normalizeRunCard(input RunCard) RunCard {
	out := RunCard{
		CardID:          strings.TrimSpace(input.CardID),
		RunID:           strings.TrimSpace(input.RunID),
		Kind:            strings.TrimSpace(input.Kind),
		Title:           strings.TrimSpace(input.Title),
		NodeID:          strings.TrimSpace(input.NodeID),
		NodeType:        strings.TrimSpace(input.NodeType),
		Round:           input.Round,
		Iteration:       input.Iteration,
		BranchID:        strings.TrimSpace(input.BranchID),
		SourceSessionID: strings.TrimSpace(input.SourceSessionID),
		StartedAt:       input.StartedAt,
		Status:          strings.TrimSpace(input.Status),
		FinishedAt:      input.FinishedAt,
		Preview:         strings.TrimSpace(input.Preview),
		Error:           strings.TrimSpace(input.Error),
		FinalText:       strings.TrimSpace(input.FinalText),
		SourceEvents:    CloneRunCardSourceEvents(input.SourceEvents),
	}
	if out.Status == RunStatusCancelled {
		out.Error = ""
	}
	if !out.StartedAt.IsZero() {
		out.StartedAt = out.StartedAt.UTC()
	}
	if !out.FinishedAt.IsZero() {
		out.FinishedAt = out.FinishedAt.UTC()
	}
	return out
}
