package orchestrations

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestRunnerExecutesPlanAndCarriesUpstreamTranscript(t *testing.T) {
	groups := &recordingGroupExecutor{
		results: map[string]GroupResult{
			"group-1": successGroupResult("alpha"),
			"group-2": successGroupResult("omega"),
		},
	}
	result := runnerWithGroups(groups).Execute(context.Background(), ExecuteCommand{
		Definition: &bridgeTasks.OrchestrationDefinition{},
		TraceID:    "trace",
	})

	if result.Status != bridgeTasks.RunStatusSuccess || len(result.NodeResults) != expectedTwoGroups {
		t.Fatalf("unexpected runner result: %#v", result)
	}
	if groups.commands[1].PreviousGroupID != "group-1" {
		t.Fatalf("expected second group to receive upstream group id: %#v", groups.commands)
	}
	if !strings.Contains(groups.commands[1].PreviousTranscript.Format(), "alpha") {
		t.Fatalf("expected second group to receive upstream transcript: %#v", groups.commands[1])
	}
}

func TestRunnerStopsOnGroupFailureWithRecordedNodeResult(t *testing.T) {
	groups := &recordingGroupExecutor{
		results: map[string]GroupResult{
			"group-1": failureGroupResult("failed"),
			"group-2": successGroupResult("omega"),
		},
	}
	result := runnerWithGroups(groups).Execute(context.Background(), ExecuteCommand{})

	if result.Status != bridgeTasks.RunStatusError || len(result.NodeResults) != 1 {
		t.Fatalf("expected first failure to stop runner, got %#v", result)
	}
	if len(groups.commands) != 1 || groups.commands[0].GroupNode.ID != "group-1" {
		t.Fatalf("expected only first group to execute, got %#v", groups.commands)
	}
}

func TestRunnerAcceptsEmptyPlan(t *testing.T) {
	result := Runner{
		Planner: staticPlanner{plan: group.Plan{}},
		Groups:  &recordingGroupExecutor{},
		Mapper:  ResultMapper{},
	}.Execute(context.Background(), ExecuteCommand{})

	if result.Status != bridgeTasks.RunStatusSuccess || len(result.NodeResults) != 0 {
		t.Fatalf("unexpected empty plan result: %#v", result)
	}
}

const expectedTwoGroups = 2

type staticPlanner struct {
	plan group.Plan
	err  error
}

func (p staticPlanner) Build(*bridgeTasks.OrchestrationDefinition) (group.Plan, error) {
	return p.plan, p.err
}

type recordingGroupExecutor struct {
	results  map[string]GroupResult
	commands []GroupExecuteCommand
}

func (e *recordingGroupExecutor) ExecuteGroup(
	_ context.Context,
	cmd GroupExecuteCommand,
) GroupResult {
	e.commands = append(e.commands, cmd)
	return e.results[cmd.GroupNode.ID]
}

func runnerWithGroups(groups *recordingGroupExecutor) Runner {
	return Runner{
		Planner: staticPlanner{plan: twoGroupPlan()},
		Groups:  groups,
		Mapper:  ResultMapper{},
	}
}

func twoGroupPlan() group.Plan {
	return group.Plan{
		Nodes: map[string]bridgeTasks.OrchestrationNode{
			"group-1": runnerGroupNode("group-1", group.SpeakingModeSequential),
			"group-2": runnerGroupNode("group-2", group.SpeakingModeSequential),
		},
		EntryGroupID: "group-1",
		ControlNext:  map[string]string{"group-1": "group-2"},
		GroupMembers: map[string][]string{
			"group-1": {"agent-1"},
			"group-2": {"agent-2"},
		},
	}
}

func runnerGroupNode(id string, mode string) bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID:   id,
		Type: bridgeTasks.OrchestrationNodeTypeGroup,
		Group: &bridgeTasks.OrchestrationGroupNode{
			Title:        id,
			SpeakingMode: mode,
			MaxRounds:    1,
		},
	}
}

func successGroupResult(content string) GroupResult {
	return GroupResult{
		Status:         bridgeTasks.RunStatusSuccess,
		Preview:        content,
		Transcript:     group.Transcript{}.AppendMember(1, "A", "agent-1", content),
		MemberSessions: map[string]string{"agent-1": "session-1"},
	}
}

func failureGroupResult(message string) GroupResult {
	return GroupResult{
		Status:         bridgeTasks.RunStatusError,
		Preview:        message,
		Error:          message,
		Transcript:     group.Transcript{},
		MemberSessions: map[string]string{},
	}
}
