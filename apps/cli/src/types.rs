// Shared CLI data structures for request/response payloads and runtime state.

use serde::Serialize;

pub use crate::envelope_generated::{
    AgentPayload, AgentSendAwaitingHumanResponse, AgentSendSuccessResponse, ApiResponse,
    AskHumanOption,
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

#[derive(Debug, serde::Deserialize, Clone)]
pub struct ConfigResponse {
    pub provider: String,
    pub provider_type: String,
    pub base_url: String,
    pub model: String,
    pub chat_path: String,
    pub api_key_set: bool,
}

#[derive(Debug, Serialize, Default)]
pub struct ConfigUpdate {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub provider: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub api_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub model: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chat_path: Option<String>,
}
