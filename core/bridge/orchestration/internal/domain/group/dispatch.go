package group

import (
	"fmt"
	"strings"
)

const (
	DispatchActionPublicOnce  = "public_once"
	DispatchActionPrivateOnce = "private_once"
	DispatchActionEndGroup    = "end_group"

	minPrivateDispatchParticipants = 2
)

type DispatchCommand struct {
	Action         string   `json:"action"`
	ParticipantIDs []string `json:"participant_ids,omitempty"`
	Order          string   `json:"order,omitempty"`
	Instruction    string   `json:"instruction,omitempty"`
}

type GroupRef struct {
	ID string
}

func NewGroupRef(node Node) GroupRef {
	return GroupRef{ID: node.ID}
}

type DispatchValidator struct{}

func (DispatchValidator) Validate(
	cmd DispatchCommand,
	group GroupRef,
	memberOrder []string,
) (DispatchCommand, error) {
	action := strings.TrimSpace(cmd.Action)
	if !isDispatchAction(action) {
		return DispatchCommand{}, fmt.Errorf("orchestration_dispatch action must be public_once|private_once|end_group")
	}
	participants := NormalizeDispatchParticipants(cmd.ParticipantIDs)
	if err := validateDispatchParticipants(participants, group, memberOrder); err != nil {
		return DispatchCommand{}, err
	}
	order := NormalizeDispatchOrder(cmd.Order)
	switch action {
	case DispatchActionPublicOnce:
		if len(participants) == 0 {
			participants = append([]string(nil), memberOrder...)
		}
	case DispatchActionPrivateOnce:
		if len(participants) < minPrivateDispatchParticipants {
			return DispatchCommand{}, fmt.Errorf("private_once requires at least 2 participant_ids")
		}
		order = SpeakingModeSequential
	case DispatchActionEndGroup:
		participants = nil
		order = ""
	}
	return DispatchCommand{
		Action:         action,
		ParticipantIDs: participants,
		Order:          order,
		Instruction:    strings.TrimSpace(cmd.Instruction),
	}, nil
}

func NormalizeDispatchParticipants(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func NormalizeDispatchOrder(raw string) string {
	if strings.TrimSpace(raw) == SpeakingModeParallel {
		return SpeakingModeParallel
	}
	return SpeakingModeSequential
}

func isDispatchAction(action string) bool {
	switch action {
	case DispatchActionPublicOnce, DispatchActionPrivateOnce, DispatchActionEndGroup:
		return true
	default:
		return false
	}
}

func validateDispatchParticipants(participants []string, group GroupRef, memberOrder []string) error {
	memberSet := make(map[string]bool, len(memberOrder))
	for _, memberID := range memberOrder {
		memberSet[memberID] = true
	}
	for _, participantID := range participants {
		if !memberSet[participantID] {
			return fmt.Errorf("participant_id %q is not a member of group %q", participantID, group.ID)
		}
	}
	return nil
}
