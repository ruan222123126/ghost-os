package tasks

import (
	"ghost-os/bridge/taskdefs"
	"strings"
)

const (
	RunCardKindAgentTask           = taskdefs.RunCardKindAgentTask
	RunCardKindWorkflowAgent       = taskdefs.RunCardKindWorkflowAgent
	RunCardKindWorkflowLLM         = taskdefs.RunCardKindWorkflowLLM
	RunCardKindOrchestrationOwner  = taskdefs.RunCardKindOrchestrationOwner
	RunCardKindOrchestrationMember = taskdefs.RunCardKindOrchestrationMember
	RunCardKindRelayRound          = taskdefs.RunCardKindRelayRound
)

type RunCard = taskdefs.RunCard

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
