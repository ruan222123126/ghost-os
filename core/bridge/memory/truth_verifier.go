package memory

import (
	"encoding/json"
	"sort"
)

// TruthVerifyResult 用于比对 live dual-write 快照与 replay 结果。
type TruthVerifyResult struct {
	Match               bool             `json:"match"`
	Live                TruthWriteResult `json:"live"`
	Replay              TruthWriteResult `json:"replay"`
	ActiveClaimIDs      []string         `json:"active_claim_ids,omitempty"`
	ReplayActiveIDs     []string         `json:"replay_active_ids,omitempty"`
	ConflictClaimIDs    []string         `json:"conflict_claim_ids,omitempty"`
	ReplayConflictIDs   []string         `json:"replay_conflict_claim_ids,omitempty"`
	SupersededClaimIDs  []string         `json:"superseded_claim_ids,omitempty"`
	ReplaySupersededIDs []string         `json:"replay_superseded_claim_ids,omitempty"`
}

// TruthVerifier 负责校验 live snapshot 与 event replay 是否一致。
type TruthVerifier struct {
	writer *TruthWriter
}

func NewTruthVerifier(writer *TruthWriter) *TruthVerifier {
	if writer == nil {
		return nil
	}
	return &TruthVerifier{writer: writer}
}

func (v *TruthVerifier) Verify() (TruthVerifyResult, error) {
	if v == nil || v.writer == nil || !v.writer.Enabled() {
		return TruthVerifyResult{Match: true}, nil
	}
	liveObjects, liveClaims := v.writer.snapshotState()
	replayObjects, replayClaims, replayResult, err := v.writer.replayState()
	if err != nil {
		return TruthVerifyResult{}, err
	}
	liveResult := truthWriteResultFromSnapshots(liveObjects, liveClaims)
	liveActive, liveConflict, liveSuperseded := truthVerifierClaimSets(liveClaims)
	replayActive, replayConflict, replaySuperseded := truthVerifierClaimSets(replayClaims)
	match := truthJSONEqual(truthSortedObjects(liveObjects), truthSortedObjects(replayObjects)) &&
		truthJSONEqual(truthSortedClaims(liveClaims), truthSortedClaims(replayClaims)) &&
		truthJSONEqual(liveActive, replayActive) &&
		truthJSONEqual(liveConflict, replayConflict) &&
		truthJSONEqual(liveSuperseded, replaySuperseded) &&
		liveResult.ObjectCount == replayResult.ObjectCount &&
		liveResult.ClaimCount == replayResult.ClaimCount &&
		liveResult.SourceRefCount == replayResult.SourceRefCount
	return TruthVerifyResult{
		Match:               match,
		Live:                liveResult,
		Replay:              replayResult,
		ActiveClaimIDs:      liveActive,
		ReplayActiveIDs:     replayActive,
		ConflictClaimIDs:    liveConflict,
		ReplayConflictIDs:   replayConflict,
		SupersededClaimIDs:  liveSuperseded,
		ReplaySupersededIDs: replaySuperseded,
	}, nil
}

func truthVerifierClaimSets(claims map[string]MemoryClaim) (active []string, conflict []string, superseded []string) {
	for _, claim := range claims {
		switch normalizeTruthClaimStatus(claim.Status) {
		case truthClaimStatusConflicted:
			conflict = append(conflict, claim.ClaimID)
		case truthClaimStatusSuperseded:
			superseded = append(superseded, claim.ClaimID)
		case truthClaimStatusActive:
			active = append(active, claim.ClaimID)
		}
	}
	sort.Strings(active)
	sort.Strings(conflict)
	sort.Strings(superseded)
	return active, conflict, superseded
}

func truthJSONEqual(left any, right any) bool {
	leftJSON, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}
