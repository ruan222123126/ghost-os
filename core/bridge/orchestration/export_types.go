package orchestration

const (
	BusActionAgentSend           = busActionAgentSend
	BusStatusSuccess             = busStatusSuccess
	BusStatusError               = busStatusError
	BusAssistantSessionEndSignal = busAssistantSessionEndSignal

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
type SessionGetParams = sessionGetParams
type SessionSidebarPartition = sessionSidebarPartition
type SessionSidebarPartitionState = sessionSidebarPartitionState
type SessionSidebarPartitionPutRequest = sessionSidebarPartitionPutRequest
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
type ToolNameParams = toolNameParams
type ToolUpdateRequest = toolUpdateRequest
type ToolPayload = toolPayload
type FindIconTemplateUploadRequest = findIconTemplateUploadRequest
type FindIconTemplateUploadPayload = findIconTemplateUploadPayload
type FindIconPreviewRequest = findIconPreviewRequest
type FindIconPreviewPayload = findIconPreviewPayload
type MousePositionRequest = mousePositionRequest
type MousePositionPayload = mousePositionPayload
type SessionPushEventType = sessionPushEventType
type SessionPushEvent = sessionPushEvent
type SessionPushHub = sessionPushHub

func ValidateBusRequest(req APIRequest) error {
	return validateBusRequest(req)
}

func NewSessionPushHub() *SessionPushHub {
	return newSessionPushHub()
}

func ResolveFindIconTemplateRootPath() (string, error) {
	return resolveFindIconTemplateRoot()
}
