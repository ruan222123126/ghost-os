// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

use anyhow::{Result, anyhow};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use serde_json::Value;

#[derive(Debug, Serialize)]
pub struct ApiRequest<TParams> {
    pub action: &'static str,
    pub params: TParams,
    pub trace_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub request_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ApiResponse<TPayload> {
    pub status: String,
    pub payload: Option<TPayload>,
    #[serde(default)]
    pub error: String,
}

impl<TPayload> ApiResponse<TPayload> {
    pub fn into_result(self) -> Result<TPayload> {
        if self.status == "success" {
            return self
                .payload
                .ok_or_else(|| anyhow!("bridge response payload is missing"));
        }
        let message = self.error.trim();
        if message.is_empty() {
            return Err(anyhow!("bridge returned an unknown error"));
        }
        Err(anyhow!(message.to_string()))
    }
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AssistantSessionEndSignal {
    pub signal: String,
    pub message: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentRequest {
    #[serde(default)]
    pub mode: Option<String>,
    #[serde(default)]
    pub message: Option<String>,
    #[serde(default)]
    pub images: Option<Vec<SessionImageContent>>,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub project_root: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AskHumanOption {
    pub label: String,
    #[serde(default)]
    pub allow_custom: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentSendSuccessResponse {
    pub message: String,
    pub session_id: String,
    pub session_ended: bool,
    #[serde(default)]
    pub mode: Option<String>,
    #[serde(default)]
    pub session_end: Option<AssistantSessionEndSignal>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentSendAwaitingHumanResponse {
    pub status: String,
    pub session_id: String,
    pub question_id: String,
    pub prompt: String,
    #[serde(default)]
    pub selection_mode: Option<String>,
    #[serde(default)]
    pub options: Option<Vec<AskHumanOption>>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentStopResponsePayload {
    pub status: String,
    pub message: String,
    #[serde(default)]
    pub session_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct HumanResponseRequest {
    pub session_id: String,
    pub question_id: String,
    pub answer: String,
    #[serde(default)]
    pub cancelled: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct HumanResponseAck {
    pub session_id: String,
    pub question_id: String,
    pub accepted: bool,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentStreamEvent {
    pub id: String,
    pub step_id: String,
    pub trace_id: String,
    #[serde(default)]
    pub session_id: Option<String>,
    pub turn: i64,
    #[serde(rename = "type")]
    pub r#type: String,
    pub payload: BTreeMap<String, Value>,
    #[serde(default)]
    pub at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionImageContent {
    #[serde(default)]
    pub path: Option<String>,
    #[serde(default)]
    pub url: Option<String>,
    #[serde(default)]
    pub mime_type: Option<String>,
    #[serde(default)]
    pub width: Option<i64>,
    #[serde(default)]
    pub height: Option<i64>,
    #[serde(default)]
    pub sha256: Option<String>,
    #[serde(default)]
    pub bytes: Option<i64>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionFileContent {
    pub artifact_id: String,
    pub name: String,
    #[serde(default)]
    pub mime_type: Option<String>,
    #[serde(default)]
    pub bytes: Option<i64>,
    #[serde(default)]
    pub sha256: Option<String>,
    pub download_url: String,
    #[serde(default)]
    pub source_path: Option<String>,
    #[serde(default)]
    pub note: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionPushEvent {
    pub id: String,
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub trace_id: Option<String>,
    pub session_id: String,
    pub payload: BTreeMap<String, Value>,
    #[serde(default)]
    pub at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentRunStartedPayload {
    #[serde(default)]
    pub session_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionContentPart {
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub text: Option<String>,
    #[serde(default)]
    pub image: Option<SessionImageContent>,
    #[serde(default)]
    pub file: Option<SessionFileContent>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentCompletionDeltaPayload {
    pub kind: String,
    #[serde(default)]
    pub text: Option<String>,
    #[serde(default)]
    pub thinking: Option<String>,
    #[serde(default)]
    pub tool_call_index: Option<i64>,
    #[serde(default)]
    pub tool_call_id: Option<String>,
    #[serde(default)]
    pub tool_name: Option<String>,
    #[serde(default)]
    pub arguments_fragment: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionToolCall {
    pub id: String,
    pub name: String,
    pub arguments: BTreeMap<String, Value>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentToolCallStartedPayload {
    #[serde(default)]
    pub tool: Option<String>,
    #[serde(default)]
    pub tool_call_id: Option<String>,
    #[serde(default)]
    pub arguments_json: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionToolResult {
    pub status: String,
    pub tool: String,
    #[serde(default)]
    pub trace_id: Option<String>,
    #[serde(default)]
    pub output: Option<String>,
    #[serde(default)]
    pub error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentToolCallFinishedPayload {
    #[serde(default)]
    pub tool: Option<String>,
    #[serde(default)]
    pub tool_call_id: Option<String>,
    #[serde(default)]
    pub status: Option<String>,
    #[serde(default)]
    pub error: Option<String>,
    #[serde(default)]
    pub output: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionHumanInteraction {
    pub question_id: String,
    pub prompt: String,
    #[serde(default)]
    pub selection_mode: Option<String>,
    #[serde(default)]
    pub options: Option<Vec<AskHumanOption>>,
    #[serde(default)]
    pub answer: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentStreamMessagePayload {
    pub text: String,
    #[serde(default)]
    pub session_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionMessage {
    pub index: i64,
    pub role: String,
    #[serde(default)]
    pub text: Option<String>,
    #[serde(default)]
    pub content: Option<Vec<SessionContentPart>>,
    #[serde(default)]
    pub tool_calls: Option<Vec<SessionToolCall>>,
    #[serde(default)]
    pub tool_result: Option<SessionToolResult>,
    #[serde(default)]
    pub human_interaction: Option<SessionHumanInteraction>,
    #[serde(default)]
    pub tool_call_id: Option<String>,
    #[serde(default)]
    pub in_progress: Option<bool>,
    #[serde(default)]
    pub thinking: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentDonePayload {
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub session_ended: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionMetadata {
    pub id: String,
    pub title: String,
    pub created_at: String,
    pub updated_at: String,
    pub message_count: i64,
    pub token_count: i64,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionSidebarPartition {
    pub id: String,
    pub name: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionSidebarPartitionState {
    pub version: i64,
    pub partitions: Vec<SessionSidebarPartition>,
    pub assignments: BTreeMap<String, String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionSidebarPartitionPutRequest {
    pub version: i64,
    pub partitions: Vec<SessionSidebarPartition>,
    pub assignments: BTreeMap<String, String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionSourceAssignment {
    pub kind: String,
    pub owner_id: String,
    pub owner_name: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionSourceResolution {
    pub assignments: BTreeMap<String, SessionSourceAssignment>,
    pub hidden_session_ids: Vec<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionMessagePage {
    pub limit: i64,
    #[serde(default)]
    pub before: Option<i64>,
    #[serde(default)]
    pub start_index: Option<i64>,
    #[serde(default)]
    pub end_index: Option<i64>,
    pub has_more_before: bool,
    #[serde(default)]
    pub next_before: Option<i64>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionTurnDraftSegment {
    pub id: String,
    pub content: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionTurnDraftTool {
    pub id: String,
    pub content: String,
    #[serde(default)]
    pub tool_input: Option<String>,
    #[serde(default)]
    pub tool_name: Option<String>,
    #[serde(default)]
    pub tool_status: Option<String>,
    #[serde(default)]
    pub tool_call_id: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentErrorPayload {
    pub message: String,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub code: Option<i64>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionTurnDraftPendingQuestion {
    pub question_id: String,
    pub prompt: String,
    #[serde(default)]
    pub selection_mode: Option<String>,
    #[serde(default)]
    pub options: Option<Vec<AskHumanOption>>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionDetail {
    pub id: String,
    pub title: String,
    pub messages: Vec<SessionMessage>,
    pub created_at: String,
    pub updated_at: String,
    pub message_count: i64,
    pub page: SessionMessagePage,
    pub token_count: i64,
    #[serde(default)]
    pub turn_draft: Option<SessionTurnDraft>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionTurnDraft {
    pub trace_id: String,
    pub turn: i64,
    pub status: String,
    #[serde(default)]
    pub error: Option<String>,
    pub pending_questions: Vec<SessionTurnDraftPendingQuestion>,
    pub assistant_segments: Vec<SessionTurnDraftSegment>,
    pub thinking_segments: Vec<SessionTurnDraftSegment>,
    pub tools: Vec<SessionTurnDraftTool>,
    pub item_order: Vec<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct BridgeConfig {
    pub provider: String,
    pub provider_type: String,
    pub base_url: String,
    pub model: String,
    pub chat_path: String,
    pub project_root: String,
    pub max_turns: i64,
    pub task_execution_timeout_ms: i64,
    pub relay_default_stop_policy: String,
    pub relay_default_max_rounds: i64,
    pub relay_default_execution_timeout_ms: i64,
    pub llm_completion_retry_count: i64,
    pub llm_completion_retry_interval_ms: i64,
    pub api_key_set: bool,
    pub model_selection_enabled: bool,
    pub session_human_log_full_enabled: bool,
    pub session_system_prompt_visible_enabled: bool,
    pub assistant_markdown_enabled: bool,
    pub tool_call_compact_output_enabled: bool,
    pub memory_mode_enabled: bool,
    pub microcompact_enabled: bool,
    pub session_title_mode: String,
    pub web_search_tavily_url: String,
    pub web_search_exa_url: String,
    pub web_search_tavily_api_key_set: bool,
    pub web_search_exa_api_key_set: bool,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionPushAssistantMessagePayload {
    pub message: String,
    pub session_ended: bool,
    #[serde(default)]
    pub session_end: Option<AssistantSessionEndSignal>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct ConfigUpdate {
    #[serde(default)]
    pub provider: Option<String>,
    #[serde(default)]
    pub api_key: Option<String>,
    #[serde(default)]
    pub base_url: Option<String>,
    #[serde(default)]
    pub model: Option<String>,
    #[serde(default)]
    pub chat_path: Option<String>,
    #[serde(default)]
    pub project_root: Option<String>,
    #[serde(default)]
    pub max_turns: Option<i64>,
    #[serde(default)]
    pub task_execution_timeout_ms: Option<i64>,
    #[serde(default)]
    pub relay_default_stop_policy: Option<String>,
    #[serde(default)]
    pub relay_default_max_rounds: Option<i64>,
    #[serde(default)]
    pub relay_default_execution_timeout_ms: Option<i64>,
    #[serde(default)]
    pub llm_completion_retry_count: Option<i64>,
    #[serde(default)]
    pub llm_completion_retry_interval_ms: Option<i64>,
    #[serde(default)]
    pub session_human_log_full_enabled: Option<bool>,
    #[serde(default)]
    pub session_system_prompt_visible_enabled: Option<bool>,
    #[serde(default)]
    pub assistant_markdown_enabled: Option<bool>,
    #[serde(default)]
    pub tool_call_compact_output_enabled: Option<bool>,
    #[serde(default)]
    pub memory_mode_enabled: Option<bool>,
    #[serde(default)]
    pub microcompact_enabled: Option<bool>,
    #[serde(default)]
    pub session_title_mode: Option<String>,
    #[serde(default)]
    pub web_search_tavily_url: Option<String>,
    #[serde(default)]
    pub web_search_exa_url: Option<String>,
    #[serde(default)]
    pub web_search_tavily_api_key: Option<String>,
    #[serde(default)]
    pub web_search_exa_api_key: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SessionPushAwaitingHumanPayload {
    pub question_id: String,
    pub prompt: String,
    #[serde(default)]
    pub selection_mode: Option<String>,
    #[serde(default)]
    pub options: Option<Vec<AskHumanOption>>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct ProviderConfig {
    pub name: String,
    #[serde(rename = "type")]
    pub r#type: String,
    pub base_url: String,
    #[serde(default)]
    pub models: Option<Vec<String>>,
    #[serde(default)]
    pub context_window_tokens: Option<i64>,
    #[serde(default)]
    pub response_reserve_tokens: Option<i64>,
    #[serde(default)]
    pub model_context_window_tokens: Option<BTreeMap<String, i64>>,
    #[serde(default)]
    pub model_response_reserve_tokens: Option<BTreeMap<String, i64>>,
    pub api_key_set: bool,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskRunCardStartedPayload {
    pub card_id: String,
    pub run_id: String,
    pub kind: String,
    #[serde(default)]
    pub title: Option<String>,
    #[serde(default)]
    pub node_id: Option<String>,
    #[serde(default)]
    pub node_type: Option<String>,
    #[serde(default)]
    pub round: Option<i64>,
    #[serde(default)]
    pub iteration: Option<i64>,
    #[serde(default)]
    pub branch_id: Option<String>,
    #[serde(default)]
    pub source_session_id: Option<String>,
    pub started_at: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct ProviderConfigInput {
    pub name: String,
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub base_url: Option<String>,
    #[serde(default)]
    pub api_key: Option<String>,
    #[serde(default)]
    pub models: Option<Vec<String>>,
    #[serde(default)]
    pub context_window_tokens: Option<i64>,
    #[serde(default)]
    pub response_reserve_tokens: Option<i64>,
    #[serde(default)]
    pub model_context_window_tokens: Option<BTreeMap<String, i64>>,
    #[serde(default)]
    pub model_response_reserve_tokens: Option<BTreeMap<String, i64>>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskRunCardEventPayload {
    pub card_id: String,
    #[serde(default)]
    pub source_session_id: Option<String>,
    pub source_event: AgentStreamEvent,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct ProviderListResponse {
    pub providers: Vec<ProviderConfig>,
    pub active_provider: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskRunCardFinishedPayload {
    pub card_id: String,
    pub status: String,
    pub finished_at: String,
    #[serde(default)]
    pub preview: Option<String>,
    #[serde(default)]
    pub error: Option<String>,
    #[serde(default)]
    pub source_session_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct SetActiveProviderRequest {
    pub name: String,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowNode {
    pub id: String,
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub start: Option<WorkflowStartNode>,
    #[serde(default)]
    pub tool: Option<WorkflowToolNode>,
    #[serde(default)]
    pub llm: Option<WorkflowLLMNode>,
    #[serde(default)]
    pub agent: Option<WorkflowAgentNode>,
    #[serde(default)]
    #[serde(rename = "if")]
    pub r#if: Option<WorkflowIfNode>,
    #[serde(default)]
    #[serde(rename = "loop")]
    pub r#loop: Option<WorkflowLoopNode>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowEdge {
    pub from_node_id: String,
    pub to_node_id: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowDefinition {
    pub nodes: Vec<WorkflowNode>,
    pub edges: Vec<WorkflowEdge>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationNode {
    pub id: String,
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub group: Option<OrchestrationGroupNode>,
    #[serde(default)]
    pub agent: Option<OrchestrationAgentNode>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationGroupNode {
    pub title: String,
    pub shared_context: String,
    pub speaking_mode: String,
    #[serde(default)]
    pub owner_agent_id: Option<String>,
    pub max_rounds: i64,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentMessageTaskCreateRequest {
    pub message: String,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub runtime_overrides: Option<TaskRuntimeOverrides>,
    #[serde(default)]
    pub agent_mode: Option<String>,
    #[serde(default)]
    pub relay: Option<TaskRelayConfig>,
    #[serde(default)]
    pub task_kind: Option<String>,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationAgentNode {
    pub title: String,
    pub message: String,
    #[serde(default)]
    pub runtime_overrides: Option<TaskRuntimeOverrides>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskRuntimeOverrides {
    #[serde(default)]
    pub provider_name: Option<String>,
    #[serde(default)]
    pub model: Option<String>,
    #[serde(default)]
    pub system_prompt: Option<String>,
    #[serde(default)]
    pub preset_id: Option<String>,
    #[serde(default)]
    pub tool_allowlist: Option<Vec<String>>,
    #[serde(default)]
    pub tool_allowlist_only: Option<bool>,
    #[serde(default)]
    pub max_turns: Option<i64>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowToolNode {
    pub tool_name: String,
    #[serde(default)]
    pub arguments: Option<BTreeMap<String, Value>>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationEdge {
    pub from_node_id: String,
    pub to_node_id: String,
    pub kind: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskRelayConfig {
    pub stop_policy: String,
    pub max_rounds: i64,
    #[serde(default)]
    pub execution_timeout_ms: Option<i64>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowLLMNode {
    pub prompt: String,
    #[serde(default)]
    pub system_prompt: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationDefinition {
    pub nodes: Vec<OrchestrationNode>,
    pub edges: Vec<OrchestrationEdge>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowAgentNode {
    pub message: String,
    #[serde(default)]
    pub runtime_overrides: Option<TaskRuntimeOverrides>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowIfNode {
    #[serde(default)]
    pub source_node_id: Option<String>,
    pub operator: String,
    #[serde(default)]
    pub value: Option<String>,
    pub true_node_id: String,
    pub false_node_id: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowTaskCreateRequest {
    pub task_kind: String,
    pub workflow: WorkflowDefinition,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationTaskCreateRequest {
    pub task_kind: String,
    pub name: String,
    pub orchestration: OrchestrationDefinition,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowLoopNode {
    pub max_iterations: i64,
    pub body_node_id: String,
    pub exit_node_id: String,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct AgentMessageTaskPayload {
    pub id: String,
    pub message: String,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub runtime_overrides: Option<TaskRuntimeOverrides>,
    #[serde(default)]
    pub agent_mode: Option<String>,
    #[serde(default)]
    pub relay: Option<TaskRelayConfig>,
    pub task_kind: String,
    pub schedule_type: String,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
    #[serde(default)]
    pub last_run_at: Option<String>,
    #[serde(default)]
    pub next_run_at: Option<String>,
    #[serde(default)]
    pub last_error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct TaskUpdateRequest {
    pub id: String,
    #[serde(default)]
    pub message: Option<String>,
    #[serde(default)]
    pub name: Option<String>,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub runtime_overrides: Option<TaskRuntimeOverrides>,
    #[serde(default)]
    pub agent_mode: Option<String>,
    #[serde(default)]
    pub relay: Option<TaskRelayConfig>,
    #[serde(default)]
    pub task_kind: Option<String>,
    #[serde(default)]
    pub workflow: Option<WorkflowDefinition>,
    #[serde(default)]
    pub orchestration: Option<OrchestrationDefinition>,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    #[serde(default)]
    pub enabled: Option<bool>,
    #[serde(default)]
    pub trace_id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowTaskPayload {
    pub id: String,
    pub task_kind: String,
    pub workflow: WorkflowDefinition,
    pub schedule_type: String,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
    #[serde(default)]
    pub last_run_at: Option<String>,
    #[serde(default)]
    pub next_run_at: Option<String>,
    #[serde(default)]
    pub last_error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct OrchestrationTaskPayload {
    pub id: String,
    pub name: String,
    pub task_kind: String,
    pub orchestration: OrchestrationDefinition,
    pub schedule_type: String,
    #[serde(default)]
    pub interval_seconds: Option<i64>,
    #[serde(default)]
    pub cron_expr: Option<String>,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
    #[serde(default)]
    pub last_run_at: Option<String>,
    #[serde(default)]
    pub next_run_at: Option<String>,
    #[serde(default)]
    pub last_error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowStartNode {
    #[serde(default)]
    pub inputs: Option<Vec<WorkflowInputVariable>>,
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct WorkflowInputVariable {
    pub name: String,
    #[serde(rename = "type")]
    pub r#type: String,
    #[serde(default)]
    pub required: Option<bool>,
    #[serde(default)]
    pub default: Option<Value>,
    #[serde(default)]
    pub description: Option<String>,
}

#[derive(Debug, Deserialize, Clone, PartialEq)]
#[serde(untagged)]
pub enum AgentPayload {
    Success(AgentSendSuccessResponse),
    AwaitingHuman(AgentSendAwaitingHumanResponse),
}

#[derive(Debug, Deserialize, Clone, PartialEq)]
#[serde(untagged)]
pub enum TaskCreateRequest {
    AgentMessage(AgentMessageTaskCreateRequest),
    Workflow(WorkflowTaskCreateRequest),
    Orchestration(OrchestrationTaskCreateRequest),
}

#[derive(Debug, Deserialize, Clone, PartialEq)]
#[serde(untagged)]
pub enum TaskPayload {
    AgentMessage(AgentMessageTaskPayload),
    Workflow(WorkflowTaskPayload),
    Orchestration(OrchestrationTaskPayload),
}
