package memory

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type HygieneService struct {
	enabled bool
	store   *HygieneStore
}

func NewHygieneService(config HygieneConfig) *HygieneService {
	service := &HygieneService{
		enabled: config.Enabled,
		store:   NewHygieneStore(config.Path),
	}
	if service.store != nil {
		if err := service.store.Load(); err != nil {
			log.Printf("[MEMORY] hygiene sidecar load failed, starting empty: path=%s err=%v", config.Path, err)
		}
	}
	return service
}

func (s *HygieneService) Enabled() bool {
	return s != nil && s.enabled && s.store != nil && strings.TrimSpace(s.store.BaseDir()) != ""
}

func (s *HygieneService) BaseDir() string {
	if s == nil || s.store == nil {
		return ""
	}
	return s.store.BaseDir()
}

func (s *HygieneService) UpsertAssessment(input HygieneAssessmentInput) error {
	if !s.Enabled() {
		return nil
	}
	if err := validateHygieneAssessmentInput(input); err != nil {
		return err
	}
	target := normalizeHygieneTarget(input.Target)
	scoredAt := hygieneScoredAtOrNow(input.ScoredAt)
	traceID := strings.TrimSpace(input.TraceID)
	taskID := strings.TrimSpace(input.TaskID)
	scope := strings.TrimSpace(input.Scope)
	if err := s.store.mutate(target, func(current HygieneRecord, exists bool) (HygieneRecord, error) {
		now := time.Now().UTC()
		record := current
		if !exists {
			key, ok := target.Key()
			if !ok {
				return HygieneRecord{}, fmt.Errorf("invalid hygiene target")
			}
			record = HygieneRecord{
				Key:       key,
				ObjectID:  target.ObjectID,
				EntryID:   target.EntryID,
				CreatedAt: now,
			}
		}
		record.ObjectID = firstNonEmpty(target.ObjectID, record.ObjectID)
		record.EntryID = firstNonEmpty(target.EntryID, record.EntryID)
		record.GarbageVotes += input.VoteDelta
		record.GarbageScore = hygieneScoreFromVotes(record.GarbageVotes)
		record.QuarantineLevel = hygieneQuarantineLevelFromVotes(record.GarbageVotes)
		record.LastScoredAt = scoredAt
		record.LastTraceID = traceID
		record.UpdatedAt = now
		record.Reasons = mergeHygieneReasons(record.Reasons, input.Reasons)
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		}
		return record, nil
	}); err != nil {
		return err
	}
	key, ok := target.Key()
	if !ok {
		return fmt.Errorf("invalid hygiene target")
	}
	return s.store.AppendScoreLog(HygieneScoreLog{
		TargetKey: targetKeyOrFallback(key),
		ObjectID:  target.ObjectID,
		EntryID:   target.EntryID,
		VoteDelta: input.VoteDelta,
		Reasons:   input.Reasons,
		TraceID:   traceID,
		TaskID:    taskID,
		Scope:     scope,
		ScoredAt:  scoredAt,
	})
}

func targetKeyOrFallback(key string) string { return strings.TrimSpace(key) }

func (s *HygieneService) Get(target HygieneTarget) (HygieneRecord, bool) {
	if !s.Enabled() {
		return HygieneRecord{}, false
	}
	return s.store.Get(target)
}

func (s *HygieneService) BatchGet(targets []HygieneTarget) map[string]HygieneRecord {
	if !s.Enabled() {
		return nil
	}
	return s.store.BatchGet(targets)
}

func (s *HygieneService) ListQuarantined(level string, limit int) []HygieneRecord {
	if !s.Enabled() {
		return nil
	}
	return s.store.ListQuarantined(level, limit)
}

func (s *HygieneService) Stats() HygieneStats {
	if !s.Enabled() {
		return HygieneStats{}
	}
	return s.store.Stats()
}
