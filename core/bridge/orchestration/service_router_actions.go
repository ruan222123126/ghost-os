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
	registerAction(service, busActionAgentSend, func(ctx context.Context, params agentParams, traceID string) (ServiceResult, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerAction(service, busActionAgentStop, func(ctx context.Context, params agentStopParams, traceID string) (ServiceResult, error) {
		return service.executeAgentStopAction(ctx, params, traceID)
	})
}

func registerConfigActions(service *bridgeService) {
	registerAction(service, busActionConfigGet, func(_ context.Context, _ map[string]any, traceID string) (ServiceResult, error) {
		return service.executeConfigGetAction(traceID)
	})
	registerAction(service, busActionConfigUpdate, func(_ context.Context, params configUpdateRequest, traceID string) (ServiceResult, error) {
		return service.executeConfigUpdateAction(params, traceID)
	})
}

func registerHumanActions(service *bridgeService) {
	registerAction(service, busActionHumanResponse, func(ctx context.Context, params humanResponseParams, traceID string) (ServiceResult, error) {
		return service.executeHumanResponseAction(ctx, params, traceID)
	})
}

func registerTaskActions(service *bridgeService) {
	registerAction(service, busActionTaskCreate, func(_ context.Context, params taskCreateParams, traceID string) (ServiceResult, error) {
		return service.executeTaskCreateActionResult(params, traceID)
	})
	registerAction(service, busActionTaskList, func(_ context.Context, params taskListParams, traceID string) (ServiceResult, error) {
		scope, err := normalizeTaskListScope(params.Scope)
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		return service.executeTaskListActionResult(scope, traceID)
	})
	registerAction(service, busActionTaskGet, func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
		return service.executeTaskGetActionResult(params, traceID)
	})
	registerAction(service, busActionTaskUpdate, func(_ context.Context, params taskUpdateParams, traceID string) (ServiceResult, error) {
		return service.executeTaskUpdateActionResult(params, traceID)
	})
	registerAction(service, busActionTaskRunNow, func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
		return service.executeTaskRunNowActionResult(params, traceID)
	})
	registerAction(service, busActionTaskLogs, func(_ context.Context, params taskLogsParams, traceID string) (ServiceResult, error) {
		return service.executeTaskLogsActionResult(params, traceID)
	})
	registerAction(service, busActionTaskDelete, func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
		return service.executeTaskDeleteActionResult(params, traceID)
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
	registerAction(service, bridgerss.ActionInboxPoll, func(ctx context.Context, params bridgerss.InboxPollParams, traceID string) (ServiceResult, error) {
		return service.executeRSSInboxPollActionResult(ctx, params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxList, func(_ context.Context, params bridgerss.InboxListParams, traceID string) (ServiceResult, error) {
		return service.executeRSSInboxListActionResult(params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxGet, func(_ context.Context, params bridgerss.InboxGetParams, traceID string) (ServiceResult, error) {
		return service.executeRSSInboxGetActionResult(params, traceID)
	})
	registerAction(service, bridgerss.ActionInboxGroups, func(_ context.Context, params bridgerss.InboxGroupsParams, traceID string) (ServiceResult, error) {
		return service.executeRSSInboxGroupsActionResult(params, traceID)
	})
	registerAction(service, bridgerss.ActionBriefingBuild, func(ctx context.Context, params bridgerss.BriefingParams, traceID string) (ServiceResult, error) {
		return service.executeRSSBriefingBuildActionResult(ctx, params, traceID)
	})
	registerAction(service, bridgerss.ActionBriefingGet, func(_ context.Context, _ map[string]any, traceID string) (ServiceResult, error) {
		return service.executeRSSBriefingGetActionResult(traceID)
	})
}

// registerAction 负责“先解码参数，再调用用例”，默认约束 action 使用领域 ServiceResult 契约。
func registerAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (ServiceResult, error)) {
	service.registerAction(action, func(ctx context.Context, rawParams json.RawMessage, traceID string) (ServiceResult, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
		}
		return handler(ctx, params, traceID)
	})
}
