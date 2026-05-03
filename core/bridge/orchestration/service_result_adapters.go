package orchestration

import (
	"context"

	bridgerss "ghost-os/bridge/rss"
	bridgeskills "ghost-os/bridge/skills"
)

func adaptLegacyResult(payload any, statusCode int, err error) (ServiceResult, error) {
	return serviceResultFromStatus(payload, statusCode, err)
}

func (s *bridgeService) executeTaskCreateActionResult(params taskCreateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskCreateAction(params, traceID))
}

func (s *bridgeService) executeTaskListActionResult(scope string, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskListAction(scope, traceID))
}

func (s *bridgeService) executeTaskGetActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskGetAction(params, traceID))
}

func (s *bridgeService) executeTaskUpdateActionResult(params taskUpdateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskUpdateAction(params, traceID))
}

func (s *bridgeService) executeTaskRunNowActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskRunNowAction(params, traceID))
}

func (s *bridgeService) executeTaskLogsActionResult(params taskLogsParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskLogsAction(params, traceID))
}

func (s *bridgeService) executeTaskDeleteActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskDeleteAction(params, traceID))
}

func (s *bridgeService) executeRSSInboxPollActionResult(
	ctx context.Context,
	params bridgerss.InboxPollParams,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSInboxPollAction(ctx, params, traceID))
}

func (s *bridgeService) executeRSSInboxListActionResult(params bridgerss.InboxListParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSInboxListAction(params, traceID))
}

func (s *bridgeService) executeRSSInboxGetActionResult(params bridgerss.InboxGetParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSInboxGetAction(params, traceID))
}

func (s *bridgeService) executeRSSInboxGroupsActionResult(
	params bridgerss.InboxGroupsParams,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSInboxGroupsAction(params, traceID))
}

func (s *bridgeService) executeRSSBriefingBuildActionResult(
	ctx context.Context,
	params bridgerss.BriefingParams,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSBriefingBuildAction(ctx, params, traceID))
}

func (s *bridgeService) executeRSSBriefingGetActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeRSSBriefingGetAction(traceID))
}

func (s *bridgeService) executeSkillListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillListAction(traceID))
}

func (s *bridgeService) executeSkillUpdateActionResult(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillUpdateAction(params, req, traceID))
}

func (s *bridgeService) executeSkillDeleteActionResult(params bridgeskills.SkillIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillDeleteAction(params, traceID))
}

func (s *bridgeService) executeToolListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolListAction(traceID))
}

func (s *bridgeService) executeToolUpdateActionResult(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolUpdateAction(params, req, traceID))
}
