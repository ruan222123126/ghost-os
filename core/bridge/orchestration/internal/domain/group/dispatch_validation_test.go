package group

import (
	"strings"
	"testing"
)

func TestDispatchValidatorRejectsInvalidParticipant(t *testing.T) {
	_, err := DispatchValidator{}.Validate(
		DispatchCommand{Action: DispatchActionPublicOnce, ParticipantIDs: []string{"agent-x"}},
		GroupRef{ID: "group-1"},
		[]string{"agent-1"},
	)
	if err == nil || !strings.Contains(err.Error(), `participant_id "agent-x" is not a member of group "group-1"`) {
		t.Fatalf("expected invalid participant error, got %v", err)
	}
}

func TestDispatchValidatorRejectsPrivateOnceSingleMember(t *testing.T) {
	_, err := DispatchValidator{}.Validate(
		DispatchCommand{Action: DispatchActionPrivateOnce, ParticipantIDs: []string{"agent-1"}},
		GroupRef{ID: "group-1"},
		[]string{"agent-1", "agent-2"},
	)
	if err == nil || !strings.Contains(err.Error(), "private_once requires at least 2 participant_ids") {
		t.Fatalf("expected private_once participant count error, got %v", err)
	}
}

func TestDispatchValidatorNormalizesEndGroup(t *testing.T) {
	got, err := DispatchValidator{}.Validate(
		DispatchCommand{
			Action:         " end_group ",
			ParticipantIDs: []string{"agent-1"},
			Order:          SpeakingModeParallel,
			Instruction:    " done ",
		},
		GroupRef{ID: "group-1"},
		[]string{"agent-1"},
	)
	if err != nil {
		t.Fatalf("validate end_group: %v", err)
	}
	if got.Action != DispatchActionEndGroup || got.Order != "" || len(got.ParticipantIDs) != 0 {
		t.Fatalf("expected end_group to clear participants and order, got %#v", got)
	}
	if got.Instruction != "done" {
		t.Fatalf("expected instruction trim to remain stable, got %#v", got)
	}
}

func TestDispatchValidatorExpandsPublicOnceParticipants(t *testing.T) {
	got, err := DispatchValidator{}.Validate(
		DispatchCommand{Action: DispatchActionPublicOnce},
		GroupRef{ID: "group-1"},
		[]string{"agent-1", "agent-2"},
	)
	if err != nil {
		t.Fatalf("validate public_once: %v", err)
	}
	if strings.Join(got.ParticipantIDs, ",") != "agent-1,agent-2" {
		t.Fatalf("expected public_once participants to expand, got %#v", got)
	}
	if got.Order != SpeakingModeSequential {
		t.Fatalf("expected default sequential order, got %#v", got)
	}
}

func TestDispatchValidatorNormalizesPrivateSendMessages(t *testing.T) {
	got, err := DispatchValidator{}.Validate(
		DispatchCommand{
			Action: DispatchActionPrivateSend,
			PrivateMessages: []PrivateMessage{{
				ParticipantID: " agent-2 ",
				Content:       " secret ",
			}},
		},
		GroupRef{ID: "group-1"},
		[]string{"agent-1", "agent-2"},
	)
	if err != nil {
		t.Fatalf("validate private_send: %v", err)
	}
	if strings.Join(got.ParticipantIDs, ",") != "agent-2" {
		t.Fatalf("expected private_send participants from messages, got %#v", got)
	}
	if len(got.PrivateMessages) != 1 || got.PrivateMessages[0].Content != "secret" {
		t.Fatalf("expected normalized private message, got %#v", got)
	}
}

func TestDispatchValidatorRejectsPrivateSendInstruction(t *testing.T) {
	_, err := DispatchValidator{}.Validate(
		DispatchCommand{
			Action:      DispatchActionPrivateSend,
			Instruction: "secret",
			PrivateMessages: []PrivateMessage{{
				ParticipantID: "agent-2",
				Content:       "secret",
			}},
		},
		GroupRef{ID: "group-1"},
		[]string{"agent-1", "agent-2"},
	)
	if err == nil || !strings.Contains(err.Error(), "private_send uses private_messages.content") {
		t.Fatalf("expected private_send instruction error, got %v", err)
	}
}

func TestDispatchValidatorRejectsDuplicatePrivateSendParticipant(t *testing.T) {
	_, err := DispatchValidator{}.Validate(
		DispatchCommand{
			Action: DispatchActionPrivateSend,
			PrivateMessages: []PrivateMessage{
				{ParticipantID: "agent-2", Content: "one"},
				{ParticipantID: "agent-2", Content: "two"},
			},
		},
		GroupRef{ID: "group-1"},
		[]string{"agent-1", "agent-2"},
	)
	if err == nil || !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("expected duplicate private_send participant error, got %v", err)
	}
}
