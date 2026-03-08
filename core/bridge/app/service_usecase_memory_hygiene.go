package app

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"ghost-os/bridge/memory"
)

var allowedMemoryHygieneReasonCodes = map[string]struct{}{
	memory.HygieneReasonDuplicateSummary:      {},
	memory.HygieneReasonToolNoise:             {},
	memory.HygieneReasonStalePlan:             {},
	memory.HygieneReasonSupersededFact:        {},
	memory.HygieneReasonLowSignalSummary:      {},
	memory.HygieneReasonTransientState:        {},
	memory.HygieneReasonRedundantDecisionMemo: {},
}

func decodeMemoryHygieneRunParams(input map[string]any) (memoryHygieneRunParams, error) {
	params, err := decodeActionParamsMap[memoryHygieneRunParams](input)
	if err != nil {
		return memoryHygieneRunParams{}, err
	}
	params.Scope = strings.TrimSpace(params.Scope)
	if params.Scope == "" {
		params.Scope = memory.HygieneScopeWarm
	}
	switch params.Scope {
	case memory.HygieneScopeWarm, memory.HygieneScopeProjection, memory.HygieneScopeReport:
	default:
		return memoryHygieneRunParams{}, fmt.Errorf("scope must be one of %s|%s|%s", memory.HygieneScopeWarm, memory.HygieneScopeProjection, memory.HygieneScopeReport)
	}
	if params.Limit <= 0 {
		return memoryHygieneRunParams{}, fmt.Errorf("limit must be > 0")
	}
	if params.MinConfidence < 0 || params.MinConfidence > 1 {
		return memoryHygieneRunParams{}, fmt.Errorf("min_confidence must be between 0 and 1")
	}
	if params.MaxVotesPerRun <= 0 {
		params.MaxVotesPerRun = 1
	}
	codes := make([]string, 0, len(params.ReasonCodes))
	seen := make(map[string]struct{}, len(params.ReasonCodes))
	for _, code := range params.ReasonCodes {
		normalized := strings.TrimSpace(code)
		if normalized == "" {
			continue
		}
		if _, ok := allowedMemoryHygieneReasonCodes[normalized]; !ok {
			return memoryHygieneRunParams{}, fmt.Errorf("unsupported reason_code %q", normalized)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		codes = append(codes, normalized)
	}
	sort.Strings(codes)
	params.ReasonCodes = codes
	if params.MinConfidence == 0 {
		params.MinConfidence = 0.5
	}
	return params, nil
}

func memoryHygieneRunParamsToMap(params memoryHygieneRunParams) map[string]any {
	out := map[string]any{
		"scope":             params.Scope,
		"limit":             params.Limit,
		"dry_run":           params.DryRun,
		"min_confidence":    params.MinConfidence,
		"max_votes_per_run": params.MaxVotesPerRun,
	}
	if len(params.ReasonCodes) > 0 {
		out["reason_codes"] = append([]string(nil), params.ReasonCodes...)
	}
	return out
}

func (s *bridgeService) executeMemoryHygieneRunAction(params memoryHygieneRunParams, traceID string) (any, int, error) {
	return s.executeMemoryHygieneRunUsecase(params, "", traceID)
}

func (s *bridgeService) executeMemoryHygieneRunUsecase(params memoryHygieneRunParams, taskID string, traceID string) (memoryHygieneRunPayload, int, error) {
	if s == nil || s.memoryManager == nil {
		return memoryHygieneRunPayload{}, http.StatusInternalServerError, fmt.Errorf("memory manager is not configured")
	}
	normalized, err := decodeMemoryHygieneRunParams(memoryHygieneRunParamsToMap(params))
	if err != nil {
		return memoryHygieneRunPayload{}, http.StatusBadRequest, err
	}
	result, err := s.memoryManager.RunHygiene(memory.HygieneRunOptions{
		Scope:          normalized.Scope,
		Limit:          normalized.Limit,
		DryRun:         normalized.DryRun,
		MinConfidence:  normalized.MinConfidence,
		MaxVotesPerRun: normalized.MaxVotesPerRun,
		ReasonCodes:    append([]string(nil), normalized.ReasonCodes...),
		TraceID:        strings.TrimSpace(traceID),
		TaskID:         strings.TrimSpace(taskID),
	})
	if err != nil {
		return memoryHygieneRunPayload{}, http.StatusInternalServerError, err
	}
	payload := memoryHygieneRunPayload{
		Scanned:               result.Scanned,
		Scored:                result.Scored,
		SuppressedCandidates:  result.SuppressedCandidates,
		QuarantinedCandidates: result.QuarantinedCandidates,
		DryRun:                result.DryRun,
		TraceID:               result.TraceID,
	}
	logAction(traceID, busActionMemoryHygieneRun, "success", nil)
	return payload, http.StatusOK, nil
}

func formatMemoryHygieneRunPreview(result memoryHygieneRunPayload) string {
	return fmt.Sprintf(
		"hygiene scanned=%d scored=%d quarantined=%d dry_run=%t",
		result.Scanned,
		result.Scored,
		result.QuarantinedCandidates,
		result.DryRun,
	)
}
