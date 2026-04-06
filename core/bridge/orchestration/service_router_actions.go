package orchestration

import (
	"context"
	"encoding/json"
	"net/http"
)

func registerDefaultActions(service *bridgeService) {
	registerAgentActions(service)
	registerConfigActions(service)
	registerHumanActions(service)
	registerTaskActions(service)
	registerRSSActions(service)
}

func registerAgentActions(service *bridgeService) {
	registerAction(service, busActionAgentSend, func(ctx context.Context, params agentParams, traceID string) (any, int, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerAction(service, busActionAgentStop, func(ctx context.Context, params agentStopParams, traceID string) (any, int, error) {
		return service.executeAgentStopAction(ctx, params, traceID)
	})
}

func registerConfigActions(service *bridgeService) {
	registerAction(service, busActionConfigGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeConfigGetAction(traceID)
	})
	registerAction(service, busActionConfigUpdate, func(_ context.Context, params configUpdateRequest, traceID string) (any, int, error) {
		return service.executeConfigUpdateAction(params, traceID)
	})
}

func registerHumanActions(service *bridgeService) {
	registerAction(service, busActionHumanResponse, func(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
		return service.executeHumanResponseAction(ctx, params, traceID)
	})
}

func registerTaskActions(service *bridgeService) {
	registerAction(service, busActionTaskCreate, func(_ context.Context, params taskCreateParams, traceID string) (any, int, error) {
		return service.executeTaskCreateAction(params, traceID)
	})
	registerAction(service, busActionTaskList, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeTaskListAction(taskListScopeUser, traceID)
	})
	registerAction(service, busActionTaskGet, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskGetAction(params, traceID)
	})
	registerAction(service, busActionTaskUpdate, func(_ context.Context, params taskUpdateParams, traceID string) (any, int, error) {
		return service.executeTaskUpdateAction(params, traceID)
	})
	registerAction(service, busActionTaskRunNow, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskRunNowAction(params, traceID)
	})
	registerAction(service, busActionTaskLogs, func(_ context.Context, params taskLogsParams, traceID string) (any, int, error) {
		return service.executeTaskLogsAction(params, traceID)
	})
	registerAction(service, busActionTaskDelete, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskDeleteAction(params, traceID)
	})
}

func registerRSSActions(service *bridgeService) {
	registerAction(service, busActionRSSInboxPoll, func(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
		return service.executeRSSInboxPollAction(ctx, params, traceID)
	})
	registerAction(service, busActionRSSInboxList, func(_ context.Context, params rssInboxListParams, traceID string) (any, int, error) {
		return service.executeRSSInboxListAction(params, traceID)
	})
	registerAction(service, busActionRSSInboxGet, func(_ context.Context, params rssInboxGetParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGetAction(params, traceID)
	})
	registerAction(service, busActionRSSInboxGroups, func(_ context.Context, params rssInboxGroupsParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGroupsAction(params, traceID)
	})
	registerAction(service, busActionRSSBriefingBuild, func(ctx context.Context, params rssBriefingParams, traceID string) (any, int, error) {
		return service.executeRSSBriefingBuildAction(ctx, params, traceID)
	})
	registerAction(service, busActionRSSBriefingGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeRSSBriefingGetAction(traceID)
	})
}

// registerAction 负责“先解码参数，再调用用例”，避免每个 action 重复样板代码。
func registerAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (any, int, error)) {
	service.actions[action] = func(ctx context.Context, rawParams json.RawMessage, traceID string) (any, int, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		return handler(ctx, params, traceID)
	}
}
