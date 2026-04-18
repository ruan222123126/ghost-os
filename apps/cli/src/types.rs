// Shared CLI data structures for request/response payloads and runtime state.

use serde::Serialize;

pub use crate::envelope_generated::{
    AgentPayload, AgentSendAwaitingHumanResponse, AgentSendSuccessResponse, ApiResponse,
    AskHumanOption, BridgeConfig, ConfigUpdate,
};

#[derive(Debug, Serialize)]
pub struct AgentParams<'a> {
    pub message: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<&'a str>,
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
            graphql_default_source: None,
            graphql_tool_runtime_enabled: None,
            graphql_text_sanitize_enabled: None,
            graphql_sources: None,
            graphql_source_upsert: None,
            graphql_mutation_policies: None,
            session_human_log_full_enabled: None,
            web_rooter_enabled: None,
            web_rooter_base_url: None,
            web_rooter_api_token: None,
            web_rooter_timeout_ms: None,
            web_search_tavily_url: None,
            web_search_exa_url: None,
            web_search_tavily_api_key: None,
            web_search_exa_api_key: None,
            trace_id: None,
        }
    }
}
