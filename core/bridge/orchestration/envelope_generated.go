// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package orchestration

import (
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

const (
	busActionAgentSend = bus.ActionAgentSend
	busActionAgentStop = bus.ActionAgentStop
	busActionHumanResponse = bus.ActionHumanResponse
	busActionConfigGet = bus.ActionConfigGet
	busActionConfigUpdate = bus.ActionConfigUpdate
	busActionTaskCreate = bus.ActionTaskCreate
	busActionTaskList = bus.ActionTaskList
	busActionTaskGet = bus.ActionTaskGet
	busActionTaskUpdate = bus.ActionTaskUpdate
	busActionTaskRunNow = bus.ActionTaskRunNow
	busActionTaskLogs = bus.ActionTaskLogs
	busActionTaskDelete = bus.ActionTaskDelete
	busActionRssInboxPoll = bus.ActionRssInboxPoll
	busActionRssInboxList = bus.ActionRssInboxList
	busActionRssInboxGet = bus.ActionRssInboxGet
	busActionRssInboxGroups = bus.ActionRssInboxGroups
	busActionRssBriefingBuild = bus.ActionRssBriefingBuild
	busActionRssBriefingGet = bus.ActionRssBriefingGet
)

const (
	busStatusSuccess = bus.StatusSuccess
	busStatusError = bus.StatusError
)

const busAssistantSessionEndSignal = bus.AssistantSessionEndSignal

type assistantSessionEndSignalPayload = api.AssistantSessionEndSignalPayload
type agentRequest = api.AgentRequest
type agentIterationSummaryItem = api.AgentIterationSummaryItem
type askHumanOption = api.AskHumanOption
type agentResponse = api.AgentResponse
type askHumanAwaitingResponse = api.AskHumanAwaitingResponse
type agentStopResponse = api.AgentStopResponse
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
type agentStreamMessagePayload = api.AgentStreamMessagePayload
type sessionMessage = api.SessionMessage
type agentDonePayload = api.AgentDonePayload
type sessionMetadata = api.SessionMetadata
type sessionSidebarPartition = api.SessionSidebarPartition
type sessionSidebarPartitionState = api.SessionSidebarPartitionState
type sessionSidebarPartitionPutRequest = api.SessionSidebarPartitionPutRequest
type sessionMessagePage = api.SessionMessagePage
type sessionTurnDraftSegment = api.SessionTurnDraftSegment
type sessionTurnDraftTool = api.SessionTurnDraftTool
type sessionTurnDraft = api.SessionTurnDraft
type agentErrorPayload = api.AgentErrorPayload
type sessionDetail = api.SessionDetail
type configResponse = api.ConfigResponse
type assistantMessagePushPayload = api.AssistantMessagePushPayload
type configUpdateRequest = api.ConfigUpdateRequest
type awaitingHumanPushPayload = api.AwaitingHumanPushPayload
type providerConfigResponse = api.ProviderConfigResponse
type providerConfigInput = api.ProviderConfigInput
type providerListResponse = api.ProviderListResponse
type setActiveProviderRequest = api.SetActiveProviderRequest
type workflowNodeContract = api.WorkflowNodeContract
type workflowEdgeContract = api.WorkflowEdgeContract
type workflowDefinitionContract = api.WorkflowDefinitionContract
type orchestrationNodeContract = api.OrchestrationNodeContract
type orchestrationGroupNodeContract = api.OrchestrationGroupNodeContract
type orchestrationAgentNodeContract = api.OrchestrationAgentNodeContract
type taskRuntimeOverridesContract = api.TaskRuntimeOverridesContract
type workflowToolNodeContract = api.WorkflowToolNodeContract
type orchestrationEdgeContract = api.OrchestrationEdgeContract
type taskRelayConfigContract = api.TaskRelayConfigContract
type workflowLLMNodeContract = api.WorkflowLLMNodeContract
type orchestrationDefinitionContract = api.OrchestrationDefinitionContract
type workflowAgentNodeContract = api.WorkflowAgentNodeContract
type workflowIfNodeContract = api.WorkflowIfNodeContract
type workflowLoopNodeContract = api.WorkflowLoopNodeContract
type workflowStartNodeContract = api.WorkflowStartNodeContract
type workflowInputVariableContract = api.WorkflowInputVariableContract

type apiRequest = bus.RequestEnvelope
type apiResponse = bus.ResponseEnvelope
