// Shared CLI data structures for request/response payloads and runtime state.

use serde::{Deserialize, Serialize};

pub use crate::envelope_generated::{ApiRequest, ApiResponse};

#[derive(Debug, Serialize)]
pub struct AgentParams<'a> {
    pub message: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<&'a str>,
}

#[derive(Debug, Deserialize)]
pub struct AgentPayload {
    #[serde(default)]
    pub message: String,
    #[serde(default)]
    pub session_id: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub question_id: String,
    #[serde(default)]
    pub prompt: String,
}

#[derive(Debug, Serialize)]
pub struct HumanResponseParams<'a> {
    pub session_id: &'a str,
    pub question_id: &'a str,
    pub answer: &'a str,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ConfigResponse {
    pub provider: String,
    pub base_url: String,
    pub model: String,
    pub chat_path: String,
    pub api_key_set: bool,
}

#[derive(Debug, Serialize, Default)]
pub struct EmptyParams {}

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
