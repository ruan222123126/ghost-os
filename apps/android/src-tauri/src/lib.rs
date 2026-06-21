use futures_util::StreamExt;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::Duration;
use tauri::{Emitter, Manager};

const BRIDGE_AGENT_STREAM_PATH: &str = "api/agent/stream";
const BRIDGE_AGENT_STREAM_CHUNK_EVENT: &str = "bridge-agent-stream-chunk";
const BRIDGE_BUS_PATH: &str = "api/bus";
const MOBILE_CONVERSATIONS_FILE: &str = "mobile-conversations.v1.json";
const REQUEST_TIMEOUT_SECS: u64 = 60;
const SSE_CONTENT_TYPE: &str = "text/event-stream";

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

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct BridgeAgentStreamCommand {
    base_url: String,
    api_token: Option<String>,
    message: String,
    request_id: String,
    session_id: Option<String>,
    trace_id: String,
}

#[derive(Serialize)]
struct BridgeBusEnvelope<'a> {
    action: &'a str,
    params: &'a Value,
    trace_id: &'a str,
}

#[derive(Serialize)]
struct BridgeAgentStreamBody<'a> {
    message: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    session_id: Option<&'a str>,
    trace_id: &'a str,
}

#[derive(Deserialize, Serialize)]
struct BridgeBusResponse {
    status: String,
    payload: Value,
    error: String,
}

#[derive(Serialize, Clone)]
#[serde(rename_all = "camelCase")]
struct BridgeAgentStreamChunk {
    request_id: String,
    chunk: Vec<u8>,
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
        return Err(format!(
            "bridge returned HTTP {status} with success envelope"
        ));
    }
    Ok(decoded)
}

#[tauri::command]
async fn bridge_agent_stream(
    app: tauri::AppHandle,
    request: BridgeAgentStreamCommand,
) -> Result<(), String> {
    validate_agent_stream_request(&request)?;
    let url = bridge_agent_stream_url(&request.base_url)?;
    let message = request.message.trim();
    let trace_id = request.trace_id.trim();
    let session_id = request
        .session_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty());
    let body = BridgeAgentStreamBody {
        message,
        session_id,
        trace_id,
    };

    let client = reqwest::Client::builder()
        .build()
        .map_err(|err| format!("create bridge stream HTTP client failed: {err}"))?;
    let mut builder = client
        .post(url)
        .header(reqwest::header::ACCEPT, SSE_CONTENT_TYPE)
        .json(&body);
    if let Some(token) = normalized_token(request.api_token.clone()) {
        builder = builder.header("X-API-Token", token);
    }

    let response = builder
        .send()
        .await
        .map_err(|err| format!("bridge stream request failed: {err}"))?;
    ensure_stream_response(response, app, request.request_id).await
}

#[tauri::command]
fn mobile_credential_save(
    app: tauri::AppHandle,
    device_id: String,
    secret: String,
) -> Result<(), String> {
    let device_id = normalize_device_id(&device_id)?;
    let secret = normalize_secret(&secret)?;
    let path = mobile_credential_path(&app)?;
    let mut credentials = read_mobile_credentials(&path)?;
    credentials.insert(device_id, secret);
    write_mobile_credentials(&path, &credentials)
}

#[tauri::command]
fn mobile_credential_load(app: tauri::AppHandle, device_id: String) -> Result<String, String> {
    let device_id = normalize_device_id(&device_id)?;
    let path = mobile_credential_path(&app)?;
    let credentials = read_mobile_credentials(&path)?;
    credentials
        .get(&device_id)
        .cloned()
        .ok_or_else(|| format!("mobile credential not found: {device_id}"))
}

#[tauri::command]
fn mobile_credential_delete(app: tauri::AppHandle, device_id: String) -> Result<(), String> {
    let device_id = normalize_device_id(&device_id)?;
    let path = mobile_credential_path(&app)?;
    let mut credentials = read_mobile_credentials(&path)?;
    credentials.remove(&device_id);
    write_mobile_credentials(&path, &credentials)
}

#[tauri::command]
fn mobile_conversations_load(app: tauri::AppHandle) -> Result<Vec<Value>, String> {
    let path = mobile_conversations_path(&app)?;
    read_mobile_conversations(&path)
}

#[tauri::command]
fn mobile_conversations_save(
    app: tauri::AppHandle,
    conversations: Vec<Value>,
) -> Result<(), String> {
    let path = mobile_conversations_path(&app)?;
    write_mobile_conversations(&path, &conversations)
}

fn bridge_bus_url(base_url: &str) -> Result<reqwest::Url, String> {
    bridge_api_url(base_url, BRIDGE_BUS_PATH)
}

fn bridge_agent_stream_url(base_url: &str) -> Result<reqwest::Url, String> {
    bridge_api_url(base_url, BRIDGE_AGENT_STREAM_PATH)
}

fn bridge_api_url(base_url: &str, path: &str) -> Result<reqwest::Url, String> {
    let trimmed = base_url.trim();
    if trimmed.is_empty() {
        return Err("bridge URL is required".to_string());
    }

    let normalized = if trimmed.ends_with('/') {
        trimmed.to_string()
    } else {
        format!("{trimmed}/")
    };
    let parsed =
        reqwest::Url::parse(&normalized).map_err(|err| format!("invalid bridge URL: {err}"))?;
    parsed
        .join(path)
        .map_err(|err| format!("build bridge URL failed: {err}"))
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

fn validate_agent_stream_request(request: &BridgeAgentStreamCommand) -> Result<(), String> {
    if request.request_id.trim().is_empty() {
        return Err("stream request_id is required".to_string());
    }
    if request.message.trim().is_empty() {
        return Err("agent message is required".to_string());
    }
    if request.trace_id.trim().is_empty() {
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

async fn ensure_stream_response(
    response: reqwest::Response,
    app: tauri::AppHandle,
    request_id: String,
) -> Result<(), String> {
    let status = response.status();
    let content_type = response
        .headers()
        .get(reqwest::header::CONTENT_TYPE)
        .and_then(|value| value.to_str().ok())
        .unwrap_or("")
        .to_string();
    if status.is_success() && is_sse_content_type(&content_type) {
        return forward_stream_chunks(response, app, request_id).await;
    }

    let body = response
        .text()
        .await
        .map_err(|err| format!("read bridge stream error response failed: {err}"))?;
    Err(parse_unexpected_stream_response_body(
        status,
        &content_type,
        &body,
    ))
}

async fn forward_stream_chunks(
    response: reqwest::Response,
    app: tauri::AppHandle,
    request_id: String,
) -> Result<(), String> {
    let mut stream = response.bytes_stream();
    while let Some(item) = stream.next().await {
        let bytes = item.map_err(|err| format!("read bridge stream failed: {err}"))?;
        app.emit(
            BRIDGE_AGENT_STREAM_CHUNK_EVENT,
            BridgeAgentStreamChunk {
                request_id: request_id.clone(),
                chunk: bytes.to_vec(),
            },
        )
        .map_err(|err| format!("emit bridge stream chunk failed: {err}"))?;
    }
    Ok(())
}

fn is_sse_content_type(content_type: &str) -> bool {
    content_type.to_ascii_lowercase().contains(SSE_CONTENT_TYPE)
}

fn parse_unexpected_stream_response_body(
    status: reqwest::StatusCode,
    content_type: &str,
    body: &str,
) -> String {
    let trimmed = body.trim();
    if trimmed.is_empty() {
        if status.is_success() && !is_sse_content_type(content_type) {
            return format!(
                "expected {SSE_CONTENT_TYPE} response but received {}",
                if content_type.trim().is_empty() {
                    "empty content-type"
                } else {
                    content_type
                }
            );
        }
        return format!("bridge stream request failed with status {status}");
    }

    if let Ok(payload) = serde_json::from_str::<Value>(trimmed) {
        if let Some(error) = payload.get("error").and_then(Value::as_str) {
            if !error.trim().is_empty() {
                return error.trim().to_string();
            }
        }
    }

    trimmed.to_string()
}

fn mobile_credential_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve mobile credential directory failed: {err}"))?;
    Ok(dir.join("mobile-credentials.json"))
}

fn mobile_conversations_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve mobile conversations directory failed: {err}"))?;
    Ok(dir.join(MOBILE_CONVERSATIONS_FILE))
}

fn read_mobile_conversations(path: &Path) -> Result<Vec<Value>, String> {
    match fs::read_to_string(path) {
        Ok(raw) => {
            if raw.trim().is_empty() {
                return Ok(Vec::new());
            }
            match serde_json::from_str::<Value>(&raw)
                .map_err(|err| format!("decode mobile conversations failed: {err}"))?
            {
                Value::Array(items) => Ok(items),
                _ => Err("mobile conversations storage must be a JSON array".to_string()),
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(Vec::new()),
        Err(err) => Err(format!("read mobile conversations failed: {err}")),
    }
}

fn write_mobile_conversations(path: &Path, conversations: &[Value]) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|err| format!("create mobile conversations directory failed: {err}"))?;
    }
    let encoded = serde_json::to_vec_pretty(conversations)
        .map_err(|err| format!("encode mobile conversations failed: {err}"))?;
    fs::write(path, encoded).map_err(|err| format!("write mobile conversations failed: {err}"))?;
    set_private_file_permissions(path, "mobile conversations")
}

fn read_mobile_credentials(path: &PathBuf) -> Result<BTreeMap<String, String>, String> {
    match fs::read_to_string(path) {
        Ok(raw) => {
            if raw.trim().is_empty() {
                Ok(BTreeMap::new())
            } else {
                serde_json::from_str(&raw)
                    .map_err(|err| format!("decode mobile credentials failed: {err}"))
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(BTreeMap::new()),
        Err(err) => Err(format!("read mobile credentials failed: {err}")),
    }
}

fn write_mobile_credentials(
    path: &PathBuf,
    credentials: &BTreeMap<String, String>,
) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|err| format!("create mobile credential directory failed: {err}"))?;
    }
    let encoded = serde_json::to_vec_pretty(credentials)
        .map_err(|err| format!("encode mobile credentials failed: {err}"))?;
    fs::write(path, encoded).map_err(|err| format!("write mobile credentials failed: {err}"))?;
    set_private_file_permissions(path, "mobile credential")
}

fn set_private_file_permissions(path: &Path, label: &str) -> Result<(), String> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;

        fs::set_permissions(path, fs::Permissions::from_mode(0o600))
            .map_err(|err| format!("set {label} permissions failed: {err}"))?;
    }
    Ok(())
}

fn normalize_device_id(raw: &str) -> Result<String, String> {
    let value = raw.trim();
    if value.is_empty() {
        return Err("device_id is required".to_string());
    }
    if !value
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || ch == '-' || ch == '_')
    {
        return Err("device_id contains unsupported characters".to_string());
    }
    Ok(value.to_string())
}

fn normalize_secret(raw: &str) -> Result<String, String> {
    let value = raw.trim();
    if value.is_empty() {
        return Err("mobile credential secret is required".to_string());
    }
    Ok(value.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bridge_agent_stream_url_joins_base_url() {
        let url = bridge_agent_stream_url("http://127.0.0.1:8711").expect("stream URL");

        assert_eq!(url.as_str(), "http://127.0.0.1:8711/api/agent/stream");
    }

    #[test]
    fn parse_unexpected_stream_response_prefers_error_field() {
        let message = parse_unexpected_stream_response_body(
            reqwest::StatusCode::BAD_REQUEST,
            "application/json",
            r#"{"status":"error","payload":{},"error":"missing model"}"#,
        );

        assert_eq!(message, "missing model");
    }

    #[test]
    fn parse_unexpected_stream_response_reports_wrong_content_type() {
        let message =
            parse_unexpected_stream_response_body(reqwest::StatusCode::OK, "application/json", "");

        assert_eq!(
            message,
            "expected text/event-stream response but received application/json"
        );
    }

    #[test]
    fn read_mobile_conversations_returns_empty_when_file_is_missing() {
        let dir = unique_test_dir("missing-conversations");
        let path = dir.join(MOBILE_CONVERSATIONS_FILE);

        let conversations = read_mobile_conversations(&path).expect("missing file loads");

        assert!(conversations.is_empty());
        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn write_mobile_conversations_creates_directory_and_writes_array() {
        let dir = unique_test_dir("write-conversations");
        let path = dir.join("nested").join(MOBILE_CONVERSATIONS_FILE);
        let payload = vec![serde_json::json!({ "id": "session-1" })];

        write_mobile_conversations(&path, &payload).expect("write conversations");

        let conversations = read_mobile_conversations(&path).expect("read conversations");
        assert_eq!(conversations, payload);
        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn read_mobile_conversations_rejects_non_array_json() {
        let dir = unique_test_dir("non-array-conversations");
        fs::create_dir_all(&dir).expect("create test dir");
        let path = dir.join(MOBILE_CONVERSATIONS_FILE);
        fs::write(&path, r#"{"id":"session-1"}"#).expect("write test file");

        let error = read_mobile_conversations(&path).expect_err("non-array JSON fails");

        assert_eq!(error, "mobile conversations storage must be a JSON array");
        let _ = fs::remove_dir_all(dir);
    }

    #[test]
    fn read_mobile_conversations_rejects_damaged_json() {
        let dir = unique_test_dir("damaged-conversations");
        fs::create_dir_all(&dir).expect("create test dir");
        let path = dir.join(MOBILE_CONVERSATIONS_FILE);
        fs::write(&path, "[").expect("write test file");

        let error = read_mobile_conversations(&path).expect_err("damaged JSON fails");

        assert!(error.starts_with("decode mobile conversations failed:"));
        let _ = fs::remove_dir_all(dir);
    }

    fn unique_test_dir(label: &str) -> PathBuf {
        let nanos = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .expect("system clock")
            .as_nanos();
        std::env::temp_dir().join(format!("ghost-os-mobile-{label}-{nanos}"))
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            host_profile,
            bridge_agent_stream,
            bridge_bus_request,
            mobile_credential_save,
            mobile_credential_load,
            mobile_credential_delete,
            mobile_conversations_load,
            mobile_conversations_save
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
