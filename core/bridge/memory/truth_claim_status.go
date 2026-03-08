package memory

import "strings"

const (
	truthClaimStatusActive     = "active"
	truthClaimStatusSuperseded = "superseded"
	truthClaimStatusConflicted = "conflicted"
	truthClaimStatusUnverified = "unverified"
)

func normalizeTruthClaimStatus(status string) string {
	switch strings.TrimSpace(status) {
	case truthClaimStatusSuperseded:
		return truthClaimStatusSuperseded
	case truthClaimStatusConflicted:
		return truthClaimStatusConflicted
	case truthClaimStatusUnverified:
		return truthClaimStatusUnverified
	default:
		return truthClaimStatusActive
	}
}

func truthClaimStatusIsLive(status string) bool {
	switch normalizeTruthClaimStatus(status) {
	case truthClaimStatusActive, truthClaimStatusConflicted, truthClaimStatusUnverified:
		return true
	default:
		return false
	}
}
