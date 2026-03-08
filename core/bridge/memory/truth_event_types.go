package memory

import "strings"

const (
	truthEventTypeEvidenceMessageObserved      = "evidence.message.observed"
	truthEventTypeEvidenceMarkdownNoteObserved = "evidence.markdown.note.observed"
	truthEventTypeEvidenceToolResultObserved   = "evidence.tool.result.observed"
	truthEventTypeClaimAsserted                = "claim.asserted"
	truthEventTypeClaimStatusChanged           = "claim.status.changed"
	truthEventTypeClaimValidityChanged         = "claim.validity.changed"
	truthEventTypeClaimRetracted               = "claim.retracted"
	truthEventTypeObjectProjected              = "object.projected"
	truthEventTypeRecipeDistilled              = "recipe.distilled"
)

func truthCanonicalEventType(eventType string, object MemoryObject, evidence MemoryEvidence) string {
	switch strings.TrimSpace(eventType) {
	case truthEventTypeEvidenceMessageObserved,
		truthEventTypeEvidenceMarkdownNoteObserved,
		truthEventTypeEvidenceToolResultObserved,
		truthEventTypeClaimAsserted,
		truthEventTypeClaimStatusChanged,
		truthEventTypeClaimValidityChanged,
		truthEventTypeClaimRetracted,
		truthEventTypeObjectProjected,
		truthEventTypeRecipeDistilled:
		return strings.TrimSpace(eventType)
	case truthEventTypeArchiveMessage:
		return truthEventTypeEvidenceMessageObserved
	case truthEventTypeMarkdownNode:
		return truthEventTypeEvidenceMarkdownNoteObserved
	case truthEventTypeDecisionMemo:
		if strings.TrimSpace(evidence.ToolCallID) != "" || strings.TrimSpace(evidence.ToolName) != "" {
			return truthEventTypeEvidenceToolResultObserved
		}
		return truthEventTypeEvidenceMessageObserved
	}
	switch {
	case strings.Contains(strings.TrimSpace(evidence.Kind), "tool"):
		return truthEventTypeEvidenceToolResultObserved
	case strings.Contains(strings.TrimSpace(evidence.Kind), "markdown"):
		return truthEventTypeEvidenceMarkdownNoteObserved
	case strings.TrimSpace(object.ObjectType) != "":
		return truthEventTypeObjectProjected
	default:
		return truthEventTypeEvidenceMessageObserved
	}
}
