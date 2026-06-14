use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::time::Duration;

const BRIDGE_BUS_PATH: &str = "api/bus";
const REQUEST_TIMEOUT_SECS: u64 = 60;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct HostProfile {
    product_name: &'static str,
    version: &'static str,
    target: &'static str,
    mobile: bool,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct BridgeBusCommand {
    base_url: String,
    api_token: Option<String>,
    action: String,
    params: Value,
    trace_id: String,
}

#[derive(Serialize)]
struct BridgeBusEnvelope<'a> {
    action: &'a str,
    params: &'a Value,
    trace_id: &'a str,
}

#[derive(Deserialize, Serialize)]
struct BridgeBusResponse {
    status: String,
    payload: Value,
    error: String,
}

#[tauri::command]
fn host_profile() -> HostProfile {
    HostProfile {
        product_name: "Ghost-OS Mobile",
        version: env!("CARGO_PKG_VERSION"),
        target: std::env::consts::OS,
        mobile: cfg!(mobile),
    }
}

#[tauri::command]
async fn bridge_bus_request(request: BridgeBusCommand) -> Result<BridgeBusResponse, String> {
    let url = bridge_bus_url(&request.base_url)?;
    let envelope = BridgeBusEnvelope {
        action: request.action.trim(),
        params: &request.params,
        trace_id: request.trace_id.trim(),
    };

    validate_envelope(&envelope)?;

    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(REQUEST_TIMEOUT_SECS))
        .build()
        .map_err(|err| format!("create bridge HTTP client failed: {err}"))?;

    let mut builder = client.post(url).json(&envelope);
    if let Some(token) = normalized_token(request.api_token) {
        builder = builder.header("X-API-Token", token);
    }

    let response = builder
        .send()
        .await
        .map_err(|err| format!("bridge request failed: {err}"))?;
    let status = response.status();
    let body = response
        .text()
        .await
        .map_err(|err| format!("read bridge response failed: {err}"))?;

    let decoded = decode_bridge_response(&body)?;
    if !status.is_success() && decoded.status == "success" {
        return Err(format!("bridge returned HTTP {status} with success envelope"));
    }
    Ok(decoded)
}

fn bridge_bus_url(base_url: &str) -> Result<reqwest::Url, String> {
    let trimmed = base_url.trim();
    if trimmed.is_empty() {
        return Err("bridge URL is required".to_string());
    }

    let normalized = if trimmed.ends_with('/') {
        trimmed.to_string()
    } else {
        format!("{trimmed}/")
    };
    let parsed = reqwest::Url::parse(&normalized)
        .map_err(|err| format!("invalid bridge URL: {err}"))?;
    parsed
        .join(BRIDGE_BUS_PATH)
        .map_err(|err| format!("build bridge bus URL failed: {err}"))
}

fn validate_envelope(envelope: &BridgeBusEnvelope<'_>) -> Result<(), String> {
    if envelope.action.is_empty() {
        return Err("bridge action is required".to_string());
    }
    if envelope.trace_id.is_empty() {
        return Err("trace_id is required".to_string());
    }
    Ok(())
}

fn normalized_token(token: Option<String>) -> Option<String> {
    let token = token?.trim().to_string();
    if token.is_empty() {
        return None;
    }
    Some(token)
}

fn decode_bridge_response(body: &str) -> Result<BridgeBusResponse, String> {
    let trimmed = body.trim();
    if trimmed.is_empty() {
        return Err("bridge returned empty response body".to_string());
    }

    let decoded: BridgeBusResponse = serde_json::from_str(trimmed)
        .map_err(|err| format!("bridge returned invalid JSON envelope: {err}"))?;
    match decoded.status.as_str() {
        "success" | "error" => Ok(decoded),
        other => Err(format!("bridge returned invalid envelope status: {other}")),
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            host_profile,
            bridge_bus_request
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
