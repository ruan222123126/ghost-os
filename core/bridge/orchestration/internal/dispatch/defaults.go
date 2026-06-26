package dispatch

import (
	"context"

	appconfig "ghost-os/bridge/orchestration/internal/app/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type TraceHandler func(context.Context, string) (bus.ServiceResult, error)

type SkillIDParams struct {
	ID string `json:"id"`
}

type SkillUpdateParams struct {
	ID      string `json:"id"`
	Enabled *bool  `json:"enabled"`
}

type DefaultHandlers struct {
	AgentSend            TypedHandler[api.AgentParams]
	AgentStop            TypedHandler[api.AgentStopParams]
	ConfigGet            TraceHandler
	ConfigUpdate         TypedHandler[api.ConfigUpdateRequest]
	ConfigProvidersGet   TraceHandler
	ConfigProviderCreate TypedHandler[api.ProviderConfigInput]
	ConfigProviderUpdate TypedHandler[api.ProviderBusUpdateRequest]
	ConfigProviderDelete TypedHandler[api.ProviderBusDeleteRequest]
	HumanResponse        TypedHandler[api.HumanResponseParams]
	SessionsList         TraceHandler
	SessionsSearch       TypedHandler[api.SessionSearchParams]
	SessionGet           TypedHandler[api.SessionGetParams]
	SessionAppend        TypedHandler[api.SessionAppendRequest]
	SkillList            TraceHandler
	SkillUpdate          TypedHandler[SkillUpdateParams]
	SkillDelete          TypedHandler[SkillIDParams]
	TaskCreate           TypedHandler[api.TaskCreateParams]
	TaskList             TypedHandler[api.TaskListParams]
	TaskGet              TypedHandler[api.TaskIDParams]
	TaskUpdate           TypedHandler[api.TaskUpdateParams]
	TaskRunNow           TypedHandler[api.TaskRunNowParams]
	TaskStop             TypedHandler[api.TaskStopParams]
	TaskLogs             TypedHandler[api.TaskLogsParams]
	TaskDelete           TypedHandler[api.TaskIDParams]
}

func RegisterDefaultActions(router *Router, handlers DefaultHandlers) {
	registerAgentActions(router, handlers)
	registerConfigActions(router, handlers)
	registerHumanActions(router, handlers)
	registerSessionActions(router, handlers)
	registerSkillActions(router, handlers)
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
	RegisterTyped(router, appconfig.ActionProviderCreate, handlers.ConfigProviderCreate)
	RegisterTyped(router, appconfig.ActionProviderUpdate, handlers.ConfigProviderUpdate)
	RegisterTyped(router, appconfig.ActionProviderDelete, handlers.ConfigProviderDelete)
}

func registerHumanActions(router *Router, handlers DefaultHandlers) {
	RegisterTyped(router, bus.ActionHumanResponse, handlers.HumanResponse)
}

func registerSessionActions(router *Router, handlers DefaultHandlers) {
	RegisterTrace(router, bus.ActionSessionsList, handlers.SessionsList)
	RegisterTyped(router, bus.ActionSessionsSearch, handlers.SessionsSearch)
	RegisterTyped(router, bus.ActionSessionGet, handlers.SessionGet)
	RegisterTyped(router, bus.ActionSessionAppend, handlers.SessionAppend)
}

func registerSkillActions(router *Router, handlers DefaultHandlers) {
	RegisterTrace(router, bus.ActionSkillList, handlers.SkillList)
	RegisterTyped(router, bus.ActionSkillUpdate, handlers.SkillUpdate)
	RegisterTyped(router, bus.ActionSkillDelete, handlers.SkillDelete)
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
