package memory

import "testing"

func TestTruthCanonicalEventType(t *testing.T) {
	object := MemoryObject{ObjectID: "obj-1", ObjectType: truthObjectTypeSemanticNote}
	if got := truthCanonicalEventType(truthEventTypeArchiveMessage, object, MemoryEvidence{Kind: "chat.message"}); got != truthEventTypeEvidenceMessageObserved {
		t.Fatalf("expected archive messages to map to evidence.message.observed, got %q", got)
	}
	if got := truthCanonicalEventType(truthEventTypeMarkdownNode, object, MemoryEvidence{Kind: "markdown.note"}); got != truthEventTypeEvidenceMarkdownNoteObserved {
		t.Fatalf("expected markdown nodes to map to markdown evidence events, got %q", got)
	}
	if got := truthCanonicalEventType(truthEventTypeDecisionMemo, object, MemoryEvidence{Kind: "tool.result", ToolName: "read_file"}); got != truthEventTypeEvidenceToolResultObserved {
		t.Fatalf("expected tool result evidence events, got %q", got)
	}
}
