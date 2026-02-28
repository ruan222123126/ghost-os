use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

use anyhow::{Context, Result, anyhow, bail};
use reqwest::StatusCode;
use reqwest::blocking::{Client as HttpClient, Response};
use serde::Serialize;
use serde::de::DeserializeOwned;
use serde_json::Value;

use crate::config::Config;
use crate::types::{
    AgentParams, AgentPayload, ApiRequest, ApiResponse, ConfigResponse, ConfigUpdate, EmptyParams,
    HumanResponseParams,
};

const ACTION_AGENT_SEND: &str = "AGENT_SEND";
const ACTION_HUMAN_RESPONSE: &str = "HUMAN_RESPONSE";
const ACTION_CONFIG_GET: &str = "CONFIG_GET";
const ACTION_CONFIG_UPDATE: &str = "CONFIG_UPDATE";

static TRACE_COUNTER: AtomicU64 = AtomicU64::new(1);

#[derive(Clone)]
pub struct BridgeClient {
    config: Config,
    http: HttpClient,
}

impl BridgeClient {
    pub fn new(config: Config) -> Result<Self> {
        let http = HttpClient::builder()
            .timeout(std::time::Duration::from_secs(config.timeout_secs))
            .build()
            .context("failed to initialize HTTP client")?;

        Ok(Self { config, http })
    }

    pub fn bridge_url(&self) -> &str {
        &self.config.bridge_url
    }

    pub fn health_check(&self) -> Result<()> {
        self.get_config().map(|_| ())
    }

    pub fn send_message(&self, message: &str, session_id: Option<&str>) -> Result<AgentPayload> {
        let trimmed = message.trim();
        let session_id = session_id.map(str::trim).filter(|value| !value.is_empty());

        if trimmed.is_empty() && session_id.is_none() {
            bail!("message is required");
        }

        self.call_bus::<_, AgentPayload>(
            ACTION_AGENT_SEND,
            &AgentParams {
                message: trimmed,
                session_id,
            },
        )
    }

    pub fn send_human_response(
        &self,
        session_id: &str,
        question_id: &str,
        answer: &str,
    ) -> Result<()> {
        let session_id = session_id.trim();
        let question_id = question_id.trim();
        let answer = answer.trim();
        if session_id.is_empty() {
            bail!("session_id is required");
        }
        if question_id.is_empty() {
            bail!("question_id is required");
        }
        if answer.is_empty() {
            bail!("answer is required");
        }

        self.call_bus::<_, Value>(
            ACTION_HUMAN_RESPONSE,
            &HumanResponseParams {
                session_id,
                question_id,
                answer,
            },
        )?;
        Ok(())
    }

    pub fn get_config(&self) -> Result<ConfigResponse> {
        self.call_bus::<_, ConfigResponse>(ACTION_CONFIG_GET, &EmptyParams::default())
    }

    pub fn update_config(&self, update: &ConfigUpdate) -> Result<ConfigResponse> {
        self.call_bus::<_, ConfigResponse>(ACTION_CONFIG_UPDATE, update)
    }

    fn call_bus<TReq, TResp>(&self, action: &'static str, params: &TReq) -> Result<TResp>
    where
        TReq: Serialize,
        TResp: DeserializeOwned,
    {
        let trace_id = next_trace_id();
        let body = ApiRequest {
            action,
            params,
            trace_id: trace_id.clone(),
        };

        let response = self
            .http
            .post(self.endpoint("/api/bus"))
            .header("X-Trace-ID", &trace_id)
            .json(&body)
            .send()
            .map_err(|error| self.map_transport_error(error, "/api/bus"))?;

        self.decode_response(response, "/api/bus")
    }

    fn endpoint(&self, path: &str) -> String {
        format!(
            "{}/{}",
            self.config.bridge_url.trim_end_matches('/'),
            path.trim_start_matches('/')
        )
    }

    fn decode_response<TResp>(&self, response: Response, path: &str) -> Result<TResp>
    where
        TResp: DeserializeOwned,
    {
        let status = response.status();
        let body = response
            .text()
            .with_context(|| format!("failed to read bridge response body for {path}"))?;
        parse_api_response(status, &body, path)
    }

    fn map_transport_error(&self, error: reqwest::Error, path: &str) -> anyhow::Error {
        if error.is_connect() {
            return anyhow!(
                "Cannot connect to Ghost-OS bridge at {}\nHint: Start the bridge with: cd core/bridge && go run . serve",
                self.config.bridge_url
            );
        }
        if error.is_timeout() {
            return anyhow!(
                "Ghost-OS bridge request timed out after {}s for {}",
                self.config.timeout_secs,
                path
            );
        }
        anyhow!("Network error while calling {path}: {error}")
    }
}

fn next_trace_id() -> String {
    let millis = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |duration| duration.as_millis());
    let sequence = TRACE_COUNTER.fetch_add(1, Ordering::Relaxed);
    format!("cli-{millis}-{sequence}")
}

fn parse_api_response<TResp>(status: StatusCode, body: &str, path: &str) -> Result<TResp>
where
    TResp: DeserializeOwned,
{
    if let Ok(api) = serde_json::from_str::<ApiResponse<TResp>>(body) {
        return api
            .into_result()
            .map_err(|error| anyhow!("bridge API error at {path}: {error}"));
    }

    // Error envelopes may contain payload shapes unrelated to TResp.
    if let Ok(api) = serde_json::from_str::<ApiResponse<Value>>(body)
        && api.status == "error"
    {
        return api
            .into_result()
            .map(|_| unreachable!("error status cannot produce success payload"))
            .map_err(|error| anyhow!("bridge API error at {path}: {error}"));
    }

    let snippet = body.lines().next().map(str::trim).unwrap_or("");
    if status.is_success() {
        bail!("invalid JSON from bridge at {path}");
    }
    if snippet.is_empty() {
        bail!("bridge request failed at {path} with HTTP {status}");
    }
    bail!("bridge request failed at {path} with HTTP {status}: {snippet}");
}

#[cfg(test)]
mod tests {
    use super::parse_api_response;
    use reqwest::StatusCode;
    use serde::Deserialize;

    #[derive(Debug, Deserialize, PartialEq)]
    struct MessagePayload {
        message: String,
    }

    #[test]
    fn parse_api_response_success_payload() {
        let body = r#"{"status":"success","payload":{"message":"ok"},"error":""}"#;
        let payload: MessagePayload = parse_api_response(StatusCode::OK, body, "/api/bus").unwrap();
        assert_eq!(
            payload,
            MessagePayload {
                message: "ok".to_string()
            }
        );
    }

    #[test]
    fn parse_api_response_api_error() {
        let body = r#"{"status":"error","payload":{},"error":"bad request"}"#;
        let err = parse_api_response::<MessagePayload>(StatusCode::BAD_REQUEST, body, "/api/bus")
            .unwrap_err();
        let text = err.to_string();
        assert!(text.contains("bridge API error"));
        assert!(text.contains("bad request"));
    }

    #[test]
    fn parse_api_response_invalid_json_on_success_http() {
        let err =
            parse_api_response::<MessagePayload>(StatusCode::OK, "{", "/api/bus").unwrap_err();
        assert!(err.to_string().contains("invalid JSON from bridge"));
    }
}
