// Shared CLI data structures for request/response payloads and runtime state.

use serde::Serialize;

pub use crate::envelope_generated::{
    AgentPayload, AgentSendAwaitingHumanResponse, AgentSendSuccessResponse, ApiResponse,
    AskHumanOption, BridgeConfig, ConfigUpdate,
};

#[derive(Debug, Serialize)]
pub struct AgentParams {
    pub message: String,
    pub project_root: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
}

impl AgentPayload {
    pub fn session_id(&self) -> &str {
        match self {
            Self::Success(payload) => &payload.session_id,
            Self::AwaitingHuman(payload) => &payload.session_id,
        }
    }

    pub fn as_awaiting_human(&self) -> Option<&AgentSendAwaitingHumanResponse> {
        match self {
            Self::AwaitingHuman(payload) => Some(payload),
            Self::Success(_) => None,
        }
    }

    pub fn into_success(self) -> Option<AgentSendSuccessResponse> {
        match self {
            Self::Success(payload) => Some(payload),
            Self::AwaitingHuman(_) => None,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct HumanResponseParams<'a> {
    pub session_id: &'a str,
    pub question_id: &'a str,
    pub answer: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cancelled: Option<bool>,
}

impl ConfigUpdate {
    pub fn empty() -> Self {
        Self {
            provider: None,
            api_key: None,
            base_url: None,
            model: None,
            chat_path: None,
            project_root: None,
            max_turns: None,
            task_execution_timeout_ms: None,
            relay_default_stop_policy: None,
            relay_default_max_rounds: None,
            relay_default_execution_timeout_ms: None,
            llm_completion_retry_count: None,
            llm_completion_retry_interval_ms: None,
            session_human_log_full_enabled: None,
            session_system_prompt_visible_enabled: None,
            assistant_markdown_enabled: None,
            tool_call_compact_output_enabled: None,
            memory_mode_enabled: None,
            microcompact_enabled: None,
            session_title_mode: None,
            web_search_tavily_url: None,
            web_search_exa_url: None,
            web_search_tavily_api_key: None,
            web_search_exa_api_key: None,
            trace_id: None,
        }
    }
}
