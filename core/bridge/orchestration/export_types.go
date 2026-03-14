package orchestration

const (
	BusActionAgentSend        = busActionAgentSend
	BusStatusSuccess          = busStatusSuccess
	BusStatusError            = busStatusError
	BusAssistantSessionEndSignal = busAssistantSessionEndSignal
	BusActionRSSInboxPoll     = busActionRSSInboxPoll
	BusActionRSSBriefingBuild = busActionRSSBriefingBuild

	DefaultMaxRequestBodyBytes = defaultMaxRequestBodyBytes
	TaskListScopeUser          = taskListScopeUser
	TaskListScopeSystem        = taskListScopeSystem

	SessionPushAssistantMessage = sessionPushAssistantMessage
	SessionPushAwaitingHuman    = sessionPushAwaitingHuman
	SessionPushRunStarted       = sessionPushRunStarted
	SessionPushCompletionDelta  = sessionPushCompletionDelta
	SessionPushToolCallStarted  = sessionPushToolCallStarted
	SessionPushToolCallFinished = sessionPushToolCallFinished
	SessionPushError            = sessionPushError
	SessionPushDone             = sessionPushDone
)

type APIRequest = apiRequest
type APIResponse = apiResponse
type AgentRequest = agentRequest
type AgentParams = agentParams
type HumanResponseParams = humanResponseParams
type SessionIDParams = sessionIDParams
type SessionDeleteResponse = sessionDeleteResponse
type ConfigResponse = configResponse
type ConfigUpdateRequest = configUpdateRequest
type ProviderCreateRequest = providerCreateRequest
type ProviderUpdateRequest = providerUpdateRequest
type ProviderConfigResponse = providerConfigResponse
type ProviderListResponse = providerListResponse
type SetActiveProviderRequest = setActiveProviderRequest
type TaskCreateParams = taskCreateParams
type TaskUpdateParams = taskUpdateParams
type TaskIDParams = taskIDParams
type TaskLogsParams = taskLogsParams
type TaskPayload = taskPayload
type TaskDeleteResponse = taskDeleteResponse
type TaskRunPayload = taskRunPayload
type TaskRunLogPayload = taskRunLogPayload
type RSSInboxPollParams = rssInboxPollParams
type RSSInboxListParams = rssInboxListParams
type RSSInboxGetParams = rssInboxGetParams
type RSSInboxGroupsParams = rssInboxGroupsParams
type RSSBriefingParams = rssBriefingParams
type SessionPushEventType = sessionPushEventType
type SessionPushEvent = sessionPushEvent
type SessionPushHub = sessionPushHub

func ValidateBusRequest(req APIRequest) error {
	return validateBusRequest(req)
}

func ValidateTaskDefinition(task *ScheduledTask) error {
	return validateTaskDefinition(task)
}

func DecodeRSSInboxPollTaskParams(input map[string]any) (RSSInboxPollParams, error) {
	return decodeRSSInboxPollParams(input)
}

func RSSInboxPollTaskParamsToMap(params RSSInboxPollParams) map[string]any {
	return rssInboxPollParamsToMap(params)
}

func DecodeRSSBriefingTaskParams(input map[string]any) (RSSBriefingParams, error) {
	return decodeRSSBriefingParams(input)
}

func RSSBriefingTaskParamsToMap(params RSSBriefingParams) map[string]any {
	return rssBriefingParamsToMap(params)
}

func NewSessionPushHub() *SessionPushHub {
	return newSessionPushHub()
}
