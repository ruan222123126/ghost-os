// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package orchestration

import (
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

const (
	busActionAgentSend            = bus.ActionAgentSend
	busActionAgentStop            = bus.ActionAgentStop
	busActionExternalAgentStart   = bus.ActionExternalAgentStart
	busActionExternalAgentSend    = bus.ActionExternalAgentSend
	busActionExternalAgentStop    = bus.ActionExternalAgentStop
	busActionExternalAgentApprove = bus.ActionExternalAgentApprove
	busActionHumanResponse        = bus.ActionHumanResponse
	busActionSessionsList         = bus.ActionSessionsList
	busActionSessionsSearch       = bus.ActionSessionsSearch
	busActionSessionGet           = bus.ActionSessionGet
	busActionSessionAppend        = bus.ActionSessionAppend
	busActionConfigGet            = bus.ActionConfigGet
	busActionConfigUpdate         = bus.ActionConfigUpdate
	busActionConfigProvidersGet   = bus.ActionConfigProvidersGet
	busActionConfigProviderExport = bus.ActionConfigProviderExport
	busActionConfigProviderCreate = bus.ActionConfigProviderCreate
	busActionConfigProviderUpdate = bus.ActionConfigProviderUpdate
	busActionConfigProviderDelete = bus.ActionConfigProviderDelete
	busActionSkillList            = bus.ActionSkillList
	busActionSkillUpdate          = bus.ActionSkillUpdate
	busActionSkillDelete          = bus.ActionSkillDelete
	busActionTaskCreate           = bus.ActionTaskCreate
	busActionTaskList             = bus.ActionTaskList
	busActionTaskGet              = bus.ActionTaskGet
	busActionTaskUpdate           = bus.ActionTaskUpdate
	busActionTaskRunNow           = bus.ActionTaskRunNow
	busActionTaskStop             = bus.ActionTaskStop
	busActionTaskLogs             = bus.ActionTaskLogs
	busActionTaskDelete           = bus.ActionTaskDelete
)

const (
	busStatusSuccess = bus.StatusSuccess
	busStatusError   = bus.StatusError
)

const busAssistantSessionEndSignal = bus.AssistantSessionEndSignal

type assistantSessionEndSignalPayload = api.AssistantSessionEndSignalPayload
type agentRequest = api.AgentRequest
type askHumanOption = api.AskHumanOption
type agentResponse = api.AgentResponse
type askHumanAwaitingResponse = api.AskHumanAwaitingResponse
type agentStopParams = api.AgentStopParams
type externalAgentRequest = api.ExternalAgentRequest
type externalAgentStopParams = api.ExternalAgentStopParams
type agentStopResponse = api.AgentStopResponse
type externalAgentApprovalParams = api.ExternalAgentApprovalParams
type externalAgentResponse = api.ExternalAgentResponse
type externalAgentApprovalResponse = api.ExternalAgentApprovalResponse
type humanResponseParams = api.HumanResponseParams
type humanResponseAck = api.HumanResponseAck
type agentStreamEventContract = api.AgentStreamEventContract
type sessionImageContent = api.SessionImageContent
type sessionFileContent = api.SessionFileContent
type sessionPushEventContract = api.SessionPushEventContract
type agentRunStartedPayload = api.AgentRunStartedPayload
type sessionContentPart = api.SessionContentPart
type agentCompletionDeltaPayload = api.AgentCompletionDeltaPayload
type sessionToolCall = api.SessionToolCall
type agentToolCallStartedPayload = api.AgentToolCallStartedPayload
type sessionToolResult = api.SessionToolResult
type agentToolCallFinishedPayload = api.AgentToolCallFinishedPayload
type sessionHumanInteraction = api.SessionHumanInteraction
type agentAwaitingHumanStreamPayload = api.AgentAwaitingHumanStreamPayload
type agentStreamMessagePayload = api.AgentStreamMessagePayload
type sessionMessage = api.SessionMessage
type agentDonePayload = api.AgentDonePayload
type sessionMetadata = api.SessionMetadata
type sessionRuntimeSelection = api.SessionRuntimeSelection
type sessionSidebarPartition = api.SessionSidebarPartition
type sessionSidebarPartitionState = api.SessionSidebarPartitionState
type sessionSidebarPartitionPutRequest = api.SessionSidebarPartitionPutRequest
type sessionSourceAssignment = api.SessionSourceAssignment
type sessionSourceResolution = api.SessionSourceResolution
type sessionMessagePage = api.SessionMessagePage
type sessionTurnDraftSegment = api.SessionTurnDraftSegment
type sessionTurnDraftTool = api.SessionTurnDraftTool
type agentErrorPayload = api.AgentErrorPayload
type sessionTurnDraftPendingQuestion = api.SessionTurnDraftPendingQuestion
type sessionDetail = api.SessionDetail
type sessionTurnDraft = api.SessionTurnDraft
type sessionAppendMessage = api.SessionAppendMessage
type sessionAppendRequest = api.SessionAppendRequest
type sessionAppendResponse = api.SessionAppendResponse
type configResponse = api.ConfigResponse
type assistantMessagePushPayload = api.AssistantMessagePushPayload
type configUpdateRequest = api.ConfigUpdateRequest
type awaitingHumanPushPayload = api.AwaitingHumanPushPayload
type providerConfigResponse = api.ProviderConfigResponse
type taskRunCardStartedPayload = api.TaskRunCardStartedPayload
type providerConfigInput = api.ProviderConfigInput
type taskRunCardEventPayload = api.TaskRunCardEventPayload
type providerSyncRecordResponse = api.ProviderSyncRecordResponse
type providerListResponse = api.ProviderListResponse
type taskRunCardFinishedPayload = api.TaskRunCardFinishedPayload
type providerBusUpdateRequest = api.ProviderBusUpdateRequest
type setActiveProviderRequest = api.SetActiveProviderRequest
type workflowNodeContract = api.WorkflowNodeContract
type workflowEdgeContract = api.WorkflowEdgeContract
type workflowDefinitionContract = api.WorkflowDefinitionContract
type orchestrationNodeContract = api.OrchestrationNodeContract
type orchestrationGroupNodeContract = api.OrchestrationGroupNodeContract
type orchestrationAgentNodeContract = api.OrchestrationAgentNodeContract
type providerExportRequest = api.ProviderExportRequest
type taskRuntimeOverridesContract = api.TaskRuntimeOverridesContract
type workflowToolNodeContract = api.WorkflowToolNodeContract
type orchestrationEdgeContract = api.OrchestrationEdgeContract
type taskRelayConfigContract = api.TaskRelayConfigContract
type workflowLLMNodeContract = api.WorkflowLLMNodeContract
type orchestrationDefinitionContract = api.OrchestrationDefinitionContract
type workflowAgentNodeContract = api.WorkflowAgentNodeContract
type workflowIfNodeContract = api.WorkflowIfNodeContract
type workflowLoopNodeContract = api.WorkflowLoopNodeContract
type providerBusDeleteRequest = api.ProviderBusDeleteRequest
type workflowStartNodeContract = api.WorkflowStartNodeContract
type workflowInputVariableContract = api.WorkflowInputVariableContract

type apiRequest = bus.RequestEnvelope
type apiResponse = bus.ResponseEnvelope

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams = api.AgentParams
type sessionIDParams = api.SessionIDParams
type sessionSearchParams = api.SessionSearchParams
type sessionGetParams = api.SessionGetParams
type sessionDeleteResponse = api.SessionDeleteResponse
type taskCreateParams = api.TaskCreateParams
type taskUpdateParams = api.TaskUpdateParams
type taskListParams = api.TaskListParams
type taskIDParams = api.TaskIDParams
type taskRunNowParams = api.TaskRunNowParams
type taskStopParams = api.TaskStopParams
type taskLogsParams = api.TaskLogsParams
type taskPayload = api.TaskPayload
type taskDeleteResponse = api.TaskDeleteResponse
type taskRunPayload = api.TaskRunPayload
type taskStopResponse = api.TaskStopResponse
type taskRunLogPayload = api.TaskRunLogPayload
type toolNameParams = api.ToolNameParams
type toolUpdateRequest = api.ToolUpdateRequest
type toolPayload = api.ToolPayload
type findIconTemplateUploadRequest = api.FindIconTemplateUploadRequest
type findIconTemplateUploadPayload = api.FindIconTemplateUploadPayload
type findIconPreviewRegion = api.FindIconPreviewRegion
type findIconPreviewRequest = api.FindIconPreviewRequest
type findIconPreviewPayload = api.FindIconPreviewPayload
type mousePositionRequest = api.MousePositionRequest
type mousePositionPayload = api.MousePositionPayload

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput

const (
	BusActionAgentSend            = busActionAgentSend
	BusActionExternalAgentStart   = busActionExternalAgentStart
	BusActionExternalAgentSend    = busActionExternalAgentSend
	BusActionExternalAgentStop    = busActionExternalAgentStop
	BusActionExternalAgentApprove = busActionExternalAgentApprove
	BusStatusSuccess              = busStatusSuccess
	BusStatusError                = busStatusError
	BusAssistantSessionEndSignal  = busAssistantSessionEndSignal

	DefaultMaxRequestBodyBytes = defaultMaxRequestBodyBytes
	TaskListScopeUser          = taskListScopeUser
	TaskListScopeSystem        = taskListScopeSystem
	TaskListScopeOrchestration = taskListScopeOrchestration

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
type ExternalAgentRequest = externalAgentRequest
type ExternalAgentStopParams = externalAgentStopParams
type ExternalAgentApprovalParams = externalAgentApprovalParams
type HumanResponseParams = humanResponseParams
type SessionIDParams = sessionIDParams
type SessionSearchParams = sessionSearchParams
type SessionGetParams = sessionGetParams
type SessionSidebarPartition = sessionSidebarPartition
type SessionSidebarPartitionState = sessionSidebarPartitionState
type SessionSidebarPartitionPutRequest = sessionSidebarPartitionPutRequest
type SessionSourceAssignment = sessionSourceAssignment
type SessionSourceResolution = sessionSourceResolution
type SessionDeleteResponse = sessionDeleteResponse
type ConfigResponse = configResponse
type ConfigUpdateRequest = configUpdateRequest
type ProviderExportRequest = providerExportRequest
type ProviderCreateRequest = providerCreateRequest
type ProviderUpdateRequest = providerUpdateRequest
type ProviderConfigResponse = providerConfigResponse
type ProviderListResponse = providerListResponse
type SetActiveProviderRequest = setActiveProviderRequest
type TaskCreateParams = taskCreateParams
type TaskUpdateParams = taskUpdateParams
type TaskListParams = taskListParams
type TaskIDParams = taskIDParams
type TaskRunNowParams = taskRunNowParams
type TaskStopParams = taskStopParams
type TaskLogsParams = taskLogsParams
type TaskPayload = taskPayload
type TaskDeleteResponse = taskDeleteResponse
type TaskRunPayload = taskRunPayload
type TaskStopResponse = taskStopResponse
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
