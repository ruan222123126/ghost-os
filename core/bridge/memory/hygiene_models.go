package memory

import (
	"fmt"
	"strings"
	"time"
)

const (
	QuarantineLevelNone   = "none"
	QuarantineLevelWeak   = "weak"
	QuarantineLevelStrong = "strong"
	QuarantineLevelHard   = "hard"
)

const (
	HygieneReasonDuplicateSummary      = "duplicate_summary"
	HygieneReasonToolNoise             = "tool_noise"
	HygieneReasonStalePlan             = "stale_plan"
	HygieneReasonSupersededFact        = "superseded_fact_projection"
	HygieneReasonLowSignalSummary      = "low_signal_summary"
	HygieneReasonTransientState        = "transient_state"
	HygieneReasonRedundantDecisionMemo = "redundant_decision_memo"
)

const defaultHygieneRecordsPathName = "records.json"

const (
	HygieneScopeWarm       = "warm"
	HygieneScopeProjection = "projection"
	HygieneScopeReport     = "report"
)

// HygieneTarget identifies a hygiene sidecar record by object first, then entry.
type HygieneTarget struct {
	ObjectID string `json:"object_id,omitempty"`
	EntryID  string `json:"entry_id,omitempty"`
}

// Key returns the canonical sidecar key for the target.
func (t HygieneTarget) Key() (string, bool) {
	normalized := normalizeHygieneTarget(t)
	if normalized.ObjectID != "" {
		return "object:" + normalized.ObjectID, true
	}
	if normalized.EntryID != "" {
		return "entry:" + normalized.EntryID, true
	}
	return "", false
}

// HygieneReason explains why a target was marked as garbage.
type HygieneReason struct {
	Code     string  `json:"code,omitempty"`
	Label    string  `json:"label,omitempty"`
	Weight   float64 `json:"weight,omitempty"`
	Evidence string  `json:"evidence,omitempty"`
}

// HygieneRecord stores the current hygiene assessment snapshot for one target.
type HygieneRecord struct {
	Key             string          `json:"key"`
	ObjectID        string          `json:"object_id,omitempty"`
	EntryID         string          `json:"entry_id,omitempty"`
	GarbageVotes    int             `json:"garbage_votes"`
	GarbageScore    float64         `json:"garbage_score"`
	QuarantineLevel string          `json:"quarantine_level"`
	LastScoredAt    time.Time       `json:"last_scored_at,omitempty"`
	Reasons         []HygieneReason `json:"reasons,omitempty"`
	LastTraceID     string          `json:"last_trace_id,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at,omitempty"`
}

// HygieneAssessmentInput is the only write payload accepted by the sidecar MVP.
type HygieneAssessmentInput struct {
	Target    HygieneTarget   `json:"target"`
	VoteDelta int             `json:"vote_delta"`
	Reasons   []HygieneReason `json:"reasons,omitempty"`
	ScoredAt  time.Time       `json:"scored_at,omitempty"`
	TraceID   string          `json:"trace_id,omitempty"`
	TaskID    string          `json:"task_id,omitempty"`
	Scope     string          `json:"scope,omitempty"`
}

type HygieneScoreLog struct {
	TargetKey  string          `json:"target_key"`
	ObjectID   string          `json:"object_id,omitempty"`
	EntryID    string          `json:"entry_id,omitempty"`
	VoteDelta  int             `json:"vote_delta"`
	Reasons    []HygieneReason `json:"reasons,omitempty"`
	TraceID    string          `json:"trace_id,omitempty"`
	TaskID     string          `json:"task_id,omitempty"`
	Scope      string          `json:"scope,omitempty"`
	ScoredAt   time.Time       `json:"scored_at,omitempty"`
	RecordedAt time.Time       `json:"recorded_at,omitempty"`
}

type HygieneRunOptions struct {
	Scope          string   `json:"scope,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	DryRun         bool     `json:"dry_run,omitempty"`
	MinConfidence  float64  `json:"min_confidence,omitempty"`
	MaxVotesPerRun int      `json:"max_votes_per_run,omitempty"`
	ReasonCodes    []string `json:"reason_codes,omitempty"`
	TraceID        string   `json:"trace_id,omitempty"`
	TaskID         string   `json:"task_id,omitempty"`
}

type HygieneRunResult struct {
	Scanned               int    `json:"scanned"`
	Scored                int    `json:"scored"`
	SuppressedCandidates  int    `json:"suppressed_candidates"`
	QuarantinedCandidates int    `json:"quarantined_candidates"`
	DryRun                bool   `json:"dry_run"`
	TraceID               string `json:"trace_id"`
}

// HygieneStats summarizes the current hygiene snapshot state.
type HygieneStats struct {
	TotalRecords       int            `json:"total_records"`
	QuarantinedRecords int            `json:"quarantined_records"`
	TotalVotes         int            `json:"total_votes"`
	Levels             map[string]int `json:"levels,omitempty"`
	LastScoredAt       time.Time      `json:"last_scored_at,omitempty"`
}

func normalizeHygieneTarget(target HygieneTarget) HygieneTarget {
	return HygieneTarget{
		ObjectID: strings.TrimSpace(target.ObjectID),
		EntryID:  strings.TrimSpace(target.EntryID),
	}
}

func normalizeHygieneReason(reason HygieneReason) HygieneReason {
	normalized := HygieneReason{
		Code:     strings.TrimSpace(reason.Code),
		Label:    strings.TrimSpace(reason.Label),
		Weight:   reason.Weight,
		Evidence: strings.TrimSpace(reason.Evidence),
	}
	if normalized.Weight < 0 {
		normalized.Weight = 0
	}
	return normalized
}

func normalizeHygieneRecord(record HygieneRecord) (HygieneRecord, bool) {
	target := normalizeHygieneTarget(HygieneTarget{ObjectID: record.ObjectID, EntryID: record.EntryID})
	key, ok := target.Key()
	if !ok {
		trimmed := strings.TrimSpace(record.Key)
		if !strings.HasPrefix(trimmed, "object:") && !strings.HasPrefix(trimmed, "entry:") {
			return HygieneRecord{}, false
		}
		key = trimmed
		switch {
		case strings.HasPrefix(key, "object:"):
			target.ObjectID = strings.TrimSpace(strings.TrimPrefix(key, "object:"))
		case strings.HasPrefix(key, "entry:"):
			target.EntryID = strings.TrimSpace(strings.TrimPrefix(key, "entry:"))
		}
	}
	reasons := make([]HygieneReason, 0, len(record.Reasons))
	for _, reason := range record.Reasons {
		normalized := normalizeHygieneReason(reason)
		if hygieneReasonEmpty(normalized) {
			continue
		}
		reasons = append(reasons, normalized)
	}
	record.Key = key
	record.ObjectID = target.ObjectID
	record.EntryID = target.EntryID
	record.GarbageVotes = max(record.GarbageVotes, 0)
	record.GarbageScore = clamp01(record.GarbageScore)
	record.QuarantineLevel = normalizeQuarantineLevel(record.QuarantineLevel)
	record.LastTraceID = strings.TrimSpace(record.LastTraceID)
	record.LastScoredAt = hygieneUTC(record.LastScoredAt)
	record.UpdatedAt = hygieneUTC(record.UpdatedAt)
	record.CreatedAt = hygieneUTC(record.CreatedAt)
	record.Reasons = reasons
	return record, true
}

func validateHygieneAssessmentInput(input HygieneAssessmentInput) error {
	if _, ok := normalizeHygieneTarget(input.Target).Key(); !ok {
		return fmt.Errorf("hygiene target requires object_id or entry_id")
	}
	if input.VoteDelta <= 0 {
		return fmt.Errorf("hygiene vote_delta must be positive")
	}
	return nil
}

func normalizeQuarantineLevel(level string) string {
	switch strings.TrimSpace(level) {
	case QuarantineLevelWeak:
		return QuarantineLevelWeak
	case QuarantineLevelStrong:
		return QuarantineLevelStrong
	case QuarantineLevelHard:
		return QuarantineLevelHard
	default:
		return QuarantineLevelNone
	}
}

func hygieneUTC(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}

func hygieneReasonEmpty(reason HygieneReason) bool {
	return reason.Code == "" && reason.Label == "" && reason.Evidence == "" && reason.Weight == 0
}
