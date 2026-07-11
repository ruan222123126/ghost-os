package orchestrations

import (
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

const mapperExpectedRounds = 1

func TestResultMapperKeepsNodeResultContract(t *testing.T) {
	recorder := sharedresult.NewNodeResultRecorder(mapperExpectedRounds)
	recorder.Record(ResultMapper{}.GroupRecord(ResultMapCommand{
		GroupNode: runnerGroupNode("group-1", group.SpeakingModeOwner),
		MemberOrder: []string{
			"agent-1",
			"agent-2",
		},
		Result: mapperGroupResult(),
	}))

	results := recorder.Snapshot()
	if len(results) != mapperExpectedRounds {
		t.Fatalf("expected one node result, got %#v", results)
	}
	output := results[0].Output.(map[string]any)
	if output["completed_rounds"] != float64(mapperExpectedRounds) {
		t.Fatalf("unexpected completed_rounds: %#v", output)
	}
	if output["owner_session_id"] != "owner-session" {
		t.Fatalf("missing owner_session_id: %#v", output)
	}
	if _, exists := output["shared_transcript"]; !exists {
		t.Fatalf("missing shared_transcript: %#v", output)
	}
	if _, exists := output["dispatch_results"]; !exists {
		t.Fatalf("missing dispatch_results: %#v", output)
	}
}

func mapperGroupResult() GroupResult {
	transcript := group.Transcript{}.AppendMember(mapperExpectedRounds, "A", "agent-1", "alpha")
	return GroupResult{
		Status:          bridgeTasks.RunStatusSuccess,
		Preview:         "alpha",
		CompletedRounds: mapperExpectedRounds,
		Transcript:      transcript,
		MemberSessions:  map[string]string{"agent-1": "session-1"},
		OwnerAgentID:    "agent-1",
		OwnerSessionID:  "owner-session",
		MemberResults: []ports.MemberResult{{
			Round:   mapperExpectedRounds,
			AgentID: "agent-1",
			Status:  bridgeTasks.RunStatusSuccess,
		}},
		DispatchResults: []DispatchResult{{
			Round:  mapperExpectedRounds,
			Action: group.DispatchActionEndGroup,
		}},
	}
}
