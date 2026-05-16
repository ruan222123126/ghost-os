package group

import (
	"fmt"
	"strings"
)

const (
	DispatchActionPublicOnce  = "public_once"
	DispatchActionPrivateOnce = "private_once"
	DispatchActionPrivateSend = "private_send"
	DispatchActionEndGroup    = "end_group"

	minPrivateDispatchParticipants = 2
)

type PrivateMessage struct {
	ParticipantID string `json:"participant_id"`
	Content       string `json:"content"`
}

type DispatchCommand struct {
	Action          string           `json:"action"`
	ParticipantIDs  []string         `json:"participant_ids,omitempty"`
	Order           string           `json:"order,omitempty"`
	Instruction     string           `json:"instruction,omitempty"`
	PrivateMessages []PrivateMessage `json:"private_messages,omitempty"`
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
		return DispatchCommand{}, fmt.Errorf("orchestration_dispatch action must be public_once|private_once|private_send|end_group")
	}
	participants := NormalizeDispatchParticipants(cmd.ParticipantIDs)
	order := NormalizeDispatchOrder(cmd.Order)
	switch action {
	case DispatchActionPublicOnce:
		if err := validateDispatchParticipants(participants, group, memberOrder); err != nil {
			return DispatchCommand{}, err
		}
		if len(participants) == 0 {
			participants = append([]string(nil), memberOrder...)
		}
	case DispatchActionPrivateOnce:
		if err := validateDispatchParticipants(participants, group, memberOrder); err != nil {
			return DispatchCommand{}, err
		}
		if len(participants) < minPrivateDispatchParticipants {
			return DispatchCommand{}, fmt.Errorf("private_once requires at least 2 participant_ids")
		}
		order = SpeakingModeSequential
	case DispatchActionPrivateSend:
		messages, ids, err := validatePrivateMessages(cmd.PrivateMessages, group, memberOrder)
		if err != nil {
			return DispatchCommand{}, err
		}
		if err := validatePrivateSendOptions(cmd, ids); err != nil {
			return DispatchCommand{}, err
		}
		return DispatchCommand{Action: action, ParticipantIDs: ids, PrivateMessages: messages}, nil
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

func validatePrivateSendOptions(cmd DispatchCommand, ids []string) error {
	participants := NormalizeDispatchParticipants(cmd.ParticipantIDs)
	if len(participants) > 0 && strings.Join(participants, ",") != strings.Join(ids, ",") {
		return fmt.Errorf("private_send participant_ids must match private_messages participant_id values")
	}
	if strings.TrimSpace(cmd.Order) != "" {
		return fmt.Errorf("private_send does not support order")
	}
	if strings.TrimSpace(cmd.Instruction) != "" {
		return fmt.Errorf("private_send uses private_messages.content, not instruction")
	}
	return nil
}

func validatePrivateMessages(
	raw []PrivateMessage,
	group GroupRef,
	memberOrder []string,
) ([]PrivateMessage, []string, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("private_send requires at least 1 private_messages item")
	}
	messages := make([]PrivateMessage, 0, len(raw))
	ids := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, item := range raw {
		message, err := normalizePrivateMessage(item)
		if err != nil {
			return nil, nil, err
		}
		if seen[message.ParticipantID] {
			return nil, nil, fmt.Errorf("private_send participant_id %q appears more than once", message.ParticipantID)
		}
		seen[message.ParticipantID] = true
		messages = append(messages, message)
		ids = append(ids, message.ParticipantID)
	}
	if err := validateDispatchParticipants(ids, group, memberOrder); err != nil {
		return nil, nil, err
	}
	return messages, ids, nil
}

func normalizePrivateMessage(raw PrivateMessage) (PrivateMessage, error) {
	participantID := strings.TrimSpace(raw.ParticipantID)
	if participantID == "" {
		return PrivateMessage{}, fmt.Errorf("private_send private_messages participant_id is required")
	}
	content := strings.TrimSpace(raw.Content)
	if content == "" {
		return PrivateMessage{}, fmt.Errorf("private_send private_messages content is required")
	}
	return PrivateMessage{ParticipantID: participantID, Content: content}, nil
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
	case DispatchActionPublicOnce, DispatchActionPrivateOnce, DispatchActionPrivateSend, DispatchActionEndGroup:
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
