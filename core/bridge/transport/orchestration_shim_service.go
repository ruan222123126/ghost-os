package transport

import (
	"context"
	"encoding/json"

	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
)

func (s *bridgeService) executeConfigGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteConfigGetAction(traceID)
}

func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteConfigUpdateAction(req, traceID)
}

func (s *bridgeService) executeProvidersGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteProvidersGetAction(traceID)
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderCreateAction(req, traceID)
}

func (s *bridgeService) executeProviderUpdateAction(name string, req providerUpdateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderUpdateAction(name, req, traceID)
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderDeleteAction(name, traceID)
}

func (s *bridgeService) executeSetActiveProviderAction(req setActiveProviderRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteSetActiveProviderAction(req, traceID)
}

func (s *bridgeService) executeSessionsListAction(traceID string) (any, int, error) {
	return s.inner.ExecuteSessionsListAction(traceID)
}

func (s *bridgeService) executeSessionGetAction(params sessionIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteSessionGetAction(params, traceID)
}

func (s *bridgeService) executeSessionDeleteAction(params sessionIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteSessionDeleteAction(params, traceID)
}

func (s *bridgeService) executeHumanAnswerAndResumeAction(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
	return s.inner.ExecuteHumanAnswerAndResumeAction(ctx, params, traceID)
}

func (s *bridgeService) executeHumanAnswerAndResumeStreamAction(ctx context.Context, params humanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.ExecuteHumanAnswerAndResumeStreamAction(ctx, params, traceID, sink)
}

func (s *bridgeService) ensureSessionNotInflight(sessionID string) (int, error) {
	return s.inner.EnsureSessionNotInflight(sessionID)
}

func (s *bridgeService) ensureSessionActive(sessionID string) (int, error) {
	return s.inner.EnsureSessionActive(sessionID)
}

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.ExecuteAgentStreamAction(ctx, params, traceID, sink)
}

func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (any, int, error) {
	return s.inner.DispatchAction(ctx, action, params, traceID)
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteRSSBriefingGetAction(traceID)
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params rssBriefingParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSBriefingBuildAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxGroupsAction(params rssInboxGroupsParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxGroupsAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxListAction(params rssInboxListParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxListAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxPollAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params rssInboxPollParams, taskID string, traceID string) (bridgeorchestration.RSSInboxPollResult, int, error) {
	return s.inner.ExecuteRSSInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *bridgeService) executeRSSInboxGetAction(params rssInboxGetParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxGetAction(params, traceID)
}

func (s *bridgeService) executeTaskListAction(scope string, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskListAction(scope, traceID)
}

func (s *bridgeService) executeTaskCreateAction(params taskCreateParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskCreateAction(params, traceID)
}

func (s *bridgeService) executeTaskLogsAction(params taskLogsParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskLogsAction(params, traceID)
}

func (s *bridgeService) executeTaskRunNowAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskRunNowAction(params, traceID)
}

func (s *bridgeService) executeTaskGetAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskGetAction(params, traceID)
}

func (s *bridgeService) executeTaskUpdateAction(params taskUpdateParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskUpdateAction(params, traceID)
}

func (s *bridgeService) executeTaskDeleteAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskDeleteAction(params, traceID)
}

func (s *bridgeService) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	return s.inner.PendingQuestionSnapshot(sessionID)
}

func requireSessionID(id string) (string, int, error) {
	return bridgeorchestration.RequireSessionID(id)
}
