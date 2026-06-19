package dispatch

import (
	"context"

	appconfig "ghost-os/bridge/orchestration/internal/app/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type TraceHandler func(context.Context, string) (bus.ServiceResult, error)

type DefaultHandlers struct {
	AgentSend          TypedHandler[api.AgentParams]
	AgentStop          TypedHandler[api.AgentStopParams]
	ConfigGet          TraceHandler
	ConfigUpdate       TypedHandler[api.ConfigUpdateRequest]
	ConfigProvidersGet TraceHandler
	HumanResponse      TypedHandler[api.HumanResponseParams]
	SessionsList       TraceHandler
	TaskCreate         TypedHandler[api.TaskCreateParams]
	TaskList           TypedHandler[api.TaskListParams]
	TaskGet            TypedHandler[api.TaskIDParams]
	TaskUpdate         TypedHandler[api.TaskUpdateParams]
	TaskRunNow         TypedHandler[api.TaskRunNowParams]
	TaskStop           TypedHandler[api.TaskStopParams]
	TaskLogs           TypedHandler[api.TaskLogsParams]
	TaskDelete         TypedHandler[api.TaskIDParams]
}

func RegisterDefaultActions(router *Router, handlers DefaultHandlers) {
	registerAgentActions(router, handlers)
	registerConfigActions(router, handlers)
	registerHumanActions(router, handlers)
	registerSessionActions(router, handlers)
	registerTaskActions(router, handlers)
}

func registerAgentActions(router *Router, handlers DefaultHandlers) {
	RegisterTyped(router, bus.ActionAgentSend, handlers.AgentSend)
	RegisterTyped(router, bus.ActionAgentStop, handlers.AgentStop)
}

func registerConfigActions(router *Router, handlers DefaultHandlers) {
	RegisterTrace(router, bus.ActionConfigGet, handlers.ConfigGet)
	RegisterTyped(router, bus.ActionConfigUpdate, handlers.ConfigUpdate)
	RegisterTrace(router, appconfig.ActionProvidersGet, handlers.ConfigProvidersGet)
}

func registerHumanActions(router *Router, handlers DefaultHandlers) {
	RegisterTyped(router, bus.ActionHumanResponse, handlers.HumanResponse)
}

func registerSessionActions(router *Router, handlers DefaultHandlers) {
	RegisterTrace(router, bus.ActionSessionsList, handlers.SessionsList)
}

func registerTaskActions(router *Router, handlers DefaultHandlers) {
	RegisterTyped(router, bus.ActionTaskCreate, handlers.TaskCreate)
	RegisterTyped(router, bus.ActionTaskList, handlers.TaskList)
	RegisterTyped(router, bus.ActionTaskGet, handlers.TaskGet)
	RegisterTyped(router, bus.ActionTaskUpdate, handlers.TaskUpdate)
	RegisterTyped(router, bus.ActionTaskRunNow, handlers.TaskRunNow)
	RegisterTyped(router, bus.ActionTaskStop, handlers.TaskStop)
	RegisterTyped(router, bus.ActionTaskLogs, handlers.TaskLogs)
	RegisterTyped(router, bus.ActionTaskDelete, handlers.TaskDelete)
}
