package transport

import "net/http"

type transportRoute struct {
	pattern string
	handler func(*transport, http.ResponseWriter, *http.Request)
}

var transportRoutes = []transportRoute{
	{pattern: "/api/bus", handler: (*transport).handleBus},
	{pattern: "/api/agent", handler: (*transport).handleAgent},
	{pattern: "/api/agent/stream", handler: (*transport).handleAgentStream},
	{pattern: "/api/external-agent/stream", handler: (*transport).handleExternalAgentStream},
	{pattern: "/api/questions/answer", handler: (*transport).handleQuestionAnswer},
	{pattern: "/api/questions/answer/stream", handler: (*transport).handleQuestionAnswerStream},
	{pattern: "/api/config", handler: (*transport).handleConfig},
	{pattern: "/api/config/providers", handler: (*transport).handleConfigProviders},
	{pattern: "/api/config/providers/export", handler: (*transport).handleConfigProviderExport},
	{pattern: "/api/config/providers/", handler: (*transport).handleConfigProviderByName},
	{pattern: "/api/config/active-provider", handler: (*transport).handleActiveProvider},
	{pattern: "/api/prompts/system", handler: (*transport).handleSystemPrompts},
	{pattern: "/api/presets", handler: (*transport).handlePresets},
	{pattern: "/api/presets/", handler: (*transport).handlePresetByID},
	{pattern: "/api/sessions", handler: (*transport).handleSessionsList},
	{pattern: "/api/sessions/search", handler: (*transport).handleSessionsSearch},
	{pattern: "/api/sessions/sources", handler: (*transport).handleSessionSources},
	{pattern: "/api/sessions/partitions", handler: (*transport).handleSessionPartitions},
	{pattern: "/api/sessions/", handler: (*transport).handleSessionByID},
	{pattern: "/api/system/tasks", handler: (*transport).handleSystemTasks},
	{pattern: "/api/orchestrations", handler: (*transport).handleOrchestrations},
	{pattern: "/api/orchestrations/", handler: (*transport).handleOrchestrationByID},
	{pattern: "/api/tasks", handler: (*transport).handleTasks},
	{pattern: "/api/tasks/", handler: (*transport).handleTaskByID},
	{pattern: "/api/skills", handler: (*transport).handleSkills},
	{pattern: "/api/skills/", handler: (*transport).handleSkillByID},
	{pattern: "/api/tools/screen/find-icon/template", handler: (*transport).handleFindIconTemplateUpload},
	{pattern: "/api/tools/screen/find-icon/preview", handler: (*transport).handleFindIconPreview},
	{pattern: "/api/tools/screen/mouse-position", handler: (*transport).handleMousePosition},
	{pattern: "/api/tools", handler: (*transport).handleTools},
	{pattern: "/api/tools/", handler: (*transport).handleToolByName},
}

func mountTransportRoutes(mux *http.ServeMux, transport *transport) {
	for _, route := range transportRoutes {
		handler := route.handler
		mux.HandleFunc(route.pattern, func(w http.ResponseWriter, r *http.Request) {
			handler(transport, w, r)
		})
	}
}
