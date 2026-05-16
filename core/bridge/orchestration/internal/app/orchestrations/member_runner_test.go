package orchestrations

import (
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestBuildGroupMemberMessageIncludesDispatchContext(t *testing.T) {
	message := BuildGroupMemberMessage(ports.MemberRunRequest{
		GroupNode: bridgeTasks.OrchestrationNode{
			Group: &bridgeTasks.OrchestrationGroupNode{
				SharedContext: "shared",
				SpeakingMode:  bridgeTasks.OrchestrationSpeakingModeOwner,
				MaxRounds:     3,
			},
		},
		MemberNode: bridgeTasks.OrchestrationNode{
			Agent: &bridgeTasks.OrchestrationAgentNode{Message: "role prompt"},
		},
		TranscriptText: "A: alpha",
		Round:          2,
		Instruction:    "focus",
		PrivateMessages: []group.PrivateMessage{{
			ParticipantID: "agent-1",
			Content:       "你的身份是预言家",
		}},
		Private: true,
	})

	for _, want := range []string{
		"成员角色提示：\nrole prompt",
		"群共享上下文：\nshared",
		"当前可见 group transcript：\nA: alpha",
		"当前可见私聊消息：\n你的身份是预言家",
		"当前轮次：2",
		"总轮次上限：3",
		"发言模式：owner",
		"回合可见性：私密",
		"本轮群主附加指令：\nfocus",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected message to contain %q, got %q", want, message)
		}
	}
}
