package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	bridgerss "ghost-os/bridge/rss"
)

func registerDefaultActions(service *bridgeService) {
	registerAgentActions(service)
	registerConfigActions(service)
	registerHumanActions(service)
	registerTaskActions(service)
	registerRSSActions(service)
}

func registerAgentActions(service *bridgeService) {
	registerResultAction(service, busActionAgentSend, func(ctx context.Context, params agentParams, traceID string) (ServiceResult, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerResultAction(service, busActionAgentStop, func(ctx context.Context, params agentStopParams, traceID string) (ServiceResult, error) {
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
	registerResultAction(service, busActionTaskList, func(_ context.Context, params taskListParams, traceID string) (ServiceResult, error) {
		scope, err := normalizeTaskListScope(params.Scope)
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		payload, code, err := service.executeTaskListAction(scope, traceID)
		return serviceResultFromLegacy(payload, code, err)
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

func normalizeTaskListScope(scope string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(scope))
	switch normalized {
	case "", taskListScopeUser:
		return taskListScopeUser, nil
	case taskListScopeSystem:
		return taskListScopeSystem, nil
	default:
		return "", fmt.Errorf("invalid task scope %q", scope)
	}
}

func registerRSSActions(service *bridgeService) {
	registerAction(service, bridgerss.ActionInboxPoll, func(ctx context.Context, params bridgerss.InboxPollParams, traceID string) (any, int, error) {
		return service.executeRSSInboxPollAction(ctx, params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxList, func(_ context.Context, params bridgerss.InboxListParams, traceID string) (any, int, error) {
		return service.executeRSSInboxListAction(params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxGet, func(_ context.Context, params bridgerss.InboxGetParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGetAction(params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxGroups, func(_ context.Context, params bridgerss.InboxGroupsParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGroupsAction(params, traceID)
	})
	registerAction(service, bridgerss.ActionBriefingBuild, func(ctx context.Context, params bridgerss.BriefingParams, traceID string) (any, int, error) {
		return service.executeRSSBriefingBuildAction(ctx, params, traceID)
	})
	registerAction(service, bridgerss.ActionBriefingGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeRSSBriefingGetAction(traceID)
	})
}

// registerAction 负责“先解码参数，再调用用例”，避免每个 action 重复样板代码。
func registerAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (any, int, error)) {
	service.registerAction(action, func(ctx context.Context, rawParams json.RawMessage, traceID string) (ServiceResult, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		payload, code, callErr := handler(ctx, params, traceID)
		return serviceResultFromLegacy(payload, code, callErr)
	})
}

func registerResultAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (ServiceResult, error)) {
	service.registerAction(action, func(ctx context.Context, rawParams json.RawMessage, traceID string) (ServiceResult, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		return handler(ctx, params, traceID)
	})
}
