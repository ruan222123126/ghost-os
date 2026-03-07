package memory

import "encoding/json"

// TruthVerifyResult 用于比对 live dual-write 快照与 replay 结果。
type TruthVerifyResult struct {
	Match  bool             `json:"match"`
	Live   TruthWriteResult `json:"live"`
	Replay TruthWriteResult `json:"replay"`
}

// TruthVerifier 只负责校验 live snapshot 与 event replay 是否一致。
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
	liveResult := TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		ObjectCount:    len(liveObjects),
		ClaimCount:     len(liveClaims),
		SourceRefCount: truthSourceRefCount(liveObjects, liveClaims),
	}
	match := truthJSONEqual(truthSortedObjects(liveObjects), truthSortedObjects(replayObjects)) &&
		truthJSONEqual(truthSortedClaims(liveClaims), truthSortedClaims(replayClaims)) &&
		liveResult.ObjectCount == replayResult.ObjectCount &&
		liveResult.ClaimCount == replayResult.ClaimCount &&
		liveResult.SourceRefCount == replayResult.SourceRefCount
	return TruthVerifyResult{
		Match:  match,
		Live:   liveResult,
		Replay: replayResult,
	}, nil
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
