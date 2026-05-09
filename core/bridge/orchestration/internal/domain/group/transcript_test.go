package group

import "testing"

func TestTranscriptInitialIncludesSharedContextAndUpstreamTranscript(t *testing.T) {
	previous := Transcript{}.AppendMember(1, "A", "agent-1", "alpha")
	got := Initial(
		Node{Group: &GroupNode{SharedContext: "shared context"}},
		"group-0",
		previous,
	)

	if len(got) != 2 {
		t.Fatalf("expected shared context and upstream entries, got %#v", got)
	}
	if got[0].Speaker != "system" || got[0].Content != "shared context" {
		t.Fatalf("unexpected shared context entry: %#v", got[0])
	}
	if got[1].Content != "上游群组 transcript:\nA: alpha" {
		t.Fatalf("unexpected upstream transcript entry: %#v", got[1])
	}
}

func TestTranscriptAppendMemberClonesInput(t *testing.T) {
	base := Transcript{{Round: 0, Speaker: "system", Content: "shared"}}
	next := base.AppendMember(1, "A", "agent-1", "alpha")

	if len(base) != 1 {
		t.Fatalf("expected base transcript to remain unchanged, got %#v", base)
	}
	if len(next) != 2 || next[1].AgentID != "agent-1" {
		t.Fatalf("unexpected appended transcript: %#v", next)
	}
}

func TestTranscriptFormatMatchesNodeResultText(t *testing.T) {
	transcript := Transcript{
		{Round: 0, Speaker: "system", Content: "shared"},
		{Round: 1, Speaker: "A", AgentID: "agent-1", Content: "alpha"},
	}

	if got := transcript.Format(); got != "system: shared\nA: alpha" {
		t.Fatalf("unexpected transcript format: %q", got)
	}
}
