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
    AgentParams, AgentPayload, ApiResponse, BridgeConfig, ConfigUpdate, HumanResponseParams,
};

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

        self.post_json::<_, AgentPayload>(
            "/api/agent",
            &self.build_agent_params(trimmed, session_id),
        )
    }

    pub fn answer_question(
        &self,
        session_id: &str,
        question_id: &str,
        answer: &str,
    ) -> Result<AgentPayload> {
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

        self.post_json::<_, AgentPayload>(
            "/api/questions/answer",
            &HumanResponseParams {
                session_id,
                question_id,
                answer,
                cancelled: None,
            },
        )
    }

    pub fn cancel_question(&self, session_id: &str, question_id: &str) -> Result<AgentPayload> {
        let session_id = session_id.trim();
        let question_id = question_id.trim();
        if session_id.is_empty() {
            bail!("session_id is required");
        }
        if question_id.is_empty() {
            bail!("question_id is required");
        }

        self.post_json::<_, AgentPayload>(
            "/api/questions/answer",
            &HumanResponseParams {
                session_id,
                question_id,
                answer: "",
                cancelled: Some(true),
            },
        )
    }

    pub fn get_config(&self) -> Result<BridgeConfig> {
        self.get_json("/api/config")
    }

    pub fn update_config(&self, update: &ConfigUpdate) -> Result<BridgeConfig> {
        self.post_json("/api/config", update)
    }

    fn get_json<TResp>(&self, path: &'static str) -> Result<TResp>
    where
        TResp: DeserializeOwned,
    {
        let trace_id = next_trace_id();
        let response = self
            .http
            .get(self.endpoint(path))
            .header("X-Trace-ID", &trace_id)
            .send()
            .map_err(|error| self.map_transport_error(error, path))?;

        self.decode_response(response, path)
    }

    fn post_json<TReq, TResp>(&self, path: &'static str, body: &TReq) -> Result<TResp>
    where
        TReq: Serialize,
        TResp: DeserializeOwned,
    {
        let trace_id = next_trace_id();
        let response = self
            .http
            .post(self.endpoint(path))
            .header("X-Trace-ID", &trace_id)
            .json(body)
            .send()
            .map_err(|error| self.map_transport_error(error, path))?;

        self.decode_response(response, path)
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

    fn build_agent_params(&self, message: &str, session_id: Option<&str>) -> AgentParams {
        AgentParams {
            message: message.to_string(),
            project_root: self.config.startup_project_root.clone(),
            session_id: session_id.map(str::to_string),
        }
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
    use crate::config::Config;
    use reqwest::StatusCode;
    use serde::Deserialize;

    #[derive(Debug, Deserialize, PartialEq)]
    struct MessagePayload {
        message: String,
    }

    #[test]
    fn parse_api_response_success_payload() {
        let body = r#"{"status":"success","payload":{"message":"ok"},"error":""}"#;
        let payload: MessagePayload =
            parse_api_response(StatusCode::OK, body, "/api/agent").unwrap();
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
        let err = parse_api_response::<MessagePayload>(StatusCode::BAD_REQUEST, body, "/api/agent")
            .unwrap_err();
        let text = err.to_string();
        assert!(text.contains("bridge API error"));
        assert!(text.contains("bad request"));
    }

    #[test]
    fn parse_api_response_invalid_json_on_success_http() {
        let err =
            parse_api_response::<MessagePayload>(StatusCode::OK, "{", "/api/agent").unwrap_err();
        assert!(err.to_string().contains("invalid JSON from bridge"));
    }

    #[test]
    fn build_agent_params_includes_startup_project_root() {
        let client = super::BridgeClient::new(Config {
            bridge_url: "http://127.0.0.1:18080".to_string(),
            timeout_secs: 3,
            startup_project_root: "/tmp/ghost-os".to_string(),
        })
        .unwrap();

        let params = client.build_agent_params("hello", Some("session-1"));

        assert_eq!(params.message, "hello");
        assert_eq!(params.project_root, "/tmp/ghost-os");
        assert_eq!(params.session_id.as_deref(), Some("session-1"));
    }
}
