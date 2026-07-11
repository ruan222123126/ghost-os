use futures_util::{
    future::{select, Either},
    pin_mut, StreamExt,
};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::Duration;
use tauri::{Emitter, Manager};

const BRIDGE_AGENT_STREAM_PATH: &str = "api/agent/stream";
const BRIDGE_EXTERNAL_AGENT_STREAM_PATH: &str = "api/external-agent/stream";
const BRIDGE_AGENT_STREAM_CHUNK_EVENT: &str = "bridge-agent-stream-chunk";
const BRIDGE_RUN_EVENTS_PATH: &str = "api/runs";
const BRIDGE_BUS_PATH: &str = "api/bus";
const MOBILE_CONVERSATIONS_FILE: &str = "mobile-conversations.v1.json";
const MOBILE_LOCAL_PROVIDERS_FILE: &str = "mobile-local-providers.v1.json";
const MOBILE_LOCAL_PROVIDER_SECRETS_FILE: &str = "mobile-local-provider-secrets.v1.json";
const REQUEST_TIMEOUT_SECS: u64 = 60;
const SSE_CONTENT_TYPE: &str = "text/event-stream";
const STREAM_IPC_FLUSH_INTERVAL_MS: u64 = 50;
const STREAM_IPC_MAX_BATCH_BYTES: usize = 16 * 1024;

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
    body: Option<Value>,
    message: String,
    path: Option<String>,
    request_id: String,
    runtime_overrides: Option<Value>,
    session_id: Option<String>,
    trace_id: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct BridgeAgentStreamReconnectCommand {
    base_url: String,
    api_token: Option<String>,
    last_event_id: Option<String>,
    request_id: String,
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
    runtime_overrides: Option<&'a Value>,
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

#[derive(Serialize, Deserialize, Clone, Default)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderState {
    providers: Vec<MobileLocalProviderRecord>,
}

#[derive(Serialize, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderRecord {
    name: String,
    #[serde(rename = "type")]
    provider_type: String,
    base_url: String,
    provider_id: String,
    updated_at: String,
    deleted_at: Option<String>,
    models: Vec<String>,
    context_window_tokens: Option<u64>,
    response_reserve_tokens: Option<u64>,
    model_context_window_tokens: Option<BTreeMap<String, u64>>,
    model_response_reserve_tokens: Option<BTreeMap<String, u64>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderListPayload {
    providers: Vec<MobileLocalProviderPayload>,
    active_provider: String,
    provider_sync_records: Vec<MobileLocalProviderPayload>,
}

#[derive(Serialize, Clone)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderPayload {
    name: String,
    #[serde(rename = "type")]
    provider_type: String,
    base_url: String,
    provider_id: String,
    updated_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    models: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    context_window_tokens: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    response_reserve_tokens: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    model_context_window_tokens: Option<BTreeMap<String, u64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    model_response_reserve_tokens: Option<BTreeMap<String, u64>>,
    api_key_set: bool,
    sync_state: &'static str,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderUpsertRequest {
    name: String,
    #[serde(rename = "type")]
    provider_type: String,
    provider_id: Option<String>,
    updated_at: Option<String>,
    deleted_at: Option<String>,
    base_url: Option<String>,
    api_key: Option<String>,
    models: Option<Vec<String>>,
    context_window_tokens: Option<u64>,
    response_reserve_tokens: Option<u64>,
    model_context_window_tokens: Option<BTreeMap<String, u64>>,
    model_response_reserve_tokens: Option<BTreeMap<String, u64>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalProviderExportPayload {
    name: String,
    #[serde(rename = "type")]
    provider_type: String,
    provider_id: String,
    updated_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    deleted_at: Option<String>,
    base_url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    api_key: Option<String>,
    models: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    context_window_tokens: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    response_reserve_tokens: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    model_context_window_tokens: Option<BTreeMap<String, u64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    model_response_reserve_tokens: Option<BTreeMap<String, u64>>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalLLMMessage {
    role: String,
    text: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalLLMSendRequest {
    history: Vec<MobileLocalLLMMessage>,
    model: String,
    provider_id: String,
    session_id: String,
    trace_id: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct MobileLocalLLMSendResponse {
    message: String,
    provider_id: String,
    model: String,
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
    let stream_path = validate_agent_stream_request(&request)?;
    let url = bridge_stream_url(&request.base_url, stream_path)?;
    let trace_id = request.trace_id.trim();
    let body = build_agent_stream_body(&request)?;

    let client = reqwest::Client::builder()
        .build()
        .map_err(|err| format!("create bridge stream HTTP client failed: {err}"))?;
    let mut builder = client
        .post(url)
        .header(reqwest::header::ACCEPT, SSE_CONTENT_TYPE)
        .header("X-Trace-ID", trace_id)
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
async fn bridge_agent_stream_reconnect(
    app: tauri::AppHandle,
    request: BridgeAgentStreamReconnectCommand,
) -> Result<(), String> {
    let trace_id = request.trace_id.trim();
    if trace_id.is_empty() {
        return Err("bridge stream trace ID is required".to_string());
    }
    if request.request_id.trim().is_empty() {
        return Err("bridge stream request ID is required".to_string());
    }

    let url = bridge_run_events_url(&request.base_url, trace_id)?;
    let client = reqwest::Client::builder()
        .build()
        .map_err(|err| format!("create bridge reconnect client failed: {err}"))?;
    let mut builder = client
        .get(url)
        .header(reqwest::header::ACCEPT, SSE_CONTENT_TYPE)
        .header("X-Trace-ID", trace_id);
    if let Some(token) = normalized_token(request.api_token) {
        builder = builder.header("X-API-Token", token);
    }
    if let Some(last_event_id) = normalized_token(request.last_event_id) {
        builder = builder.header("Last-Event-ID", last_event_id);
    }

    let response = builder
        .send()
        .await
        .map_err(|err| format!("bridge stream reconnect failed: {err}"))?;
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
async fn mobile_conversations_load(app: tauri::AppHandle) -> Result<Vec<Value>, String> {
    let path = mobile_conversations_path(&app)?;
    run_blocking_file_task("load mobile conversations", move || {
        read_mobile_conversations(&path)
    })
    .await
}

#[tauri::command]
async fn mobile_conversations_load_index(app: tauri::AppHandle) -> Result<Vec<Value>, String> {
    let path = mobile_conversations_path(&app)?;
    run_blocking_file_task("load mobile conversation index", move || {
        Ok(compact_mobile_conversations(read_mobile_conversations(
            &path,
        )?))
    })
    .await
}

#[tauri::command]
async fn mobile_conversation_get(
    app: tauri::AppHandle,
    session_id: String,
) -> Result<Option<Value>, String> {
    let path = mobile_conversations_path(&app)?;
    run_blocking_file_task("load mobile conversation", move || {
        let id = session_id.trim().to_string();
        if id.is_empty() {
            return Ok(None);
        }
        Ok(read_mobile_conversations(&path)?
            .into_iter()
            .find(|conversation| mobile_conversation_id(conversation) == Some(id.as_str())))
    })
    .await
}

#[tauri::command]
async fn mobile_conversations_upsert(
    app: tauri::AppHandle,
    conversations: Vec<Value>,
    limit: Option<usize>,
) -> Result<Vec<Value>, String> {
    let path = mobile_conversations_path(&app)?;
    run_blocking_file_task("upsert mobile conversations", move || {
        let mut current = read_mobile_conversations(&path)?;
        upsert_mobile_conversations(&mut current, conversations);
        sort_mobile_conversations(&mut current);
        if let Some(limit) = limit.filter(|value| *value > 0) {
            current.truncate(limit);
        }
        write_mobile_conversations(&path, &current)?;
        Ok(compact_mobile_conversations(current))
    })
    .await
}

#[tauri::command]
async fn mobile_conversations_save(
    app: tauri::AppHandle,
    conversations: Vec<Value>,
) -> Result<(), String> {
    let path = mobile_conversations_path(&app)?;
    run_blocking_file_task("save mobile conversations", move || {
        write_mobile_conversations(&path, &conversations)
    })
    .await
}

#[tauri::command]
fn mobile_local_provider_list(
    app: tauri::AppHandle,
) -> Result<MobileLocalProviderListPayload, String> {
    let state = read_mobile_local_provider_state(&mobile_local_providers_path(&app)?)?;
    let secrets = read_mobile_local_provider_secrets(&mobile_local_provider_secrets_path(&app)?)?;
    Ok(build_mobile_local_provider_payload(state, &secrets))
}

#[tauri::command]
fn mobile_local_provider_upsert(
    app: tauri::AppHandle,
    provider: MobileLocalProviderUpsertRequest,
) -> Result<MobileLocalProviderListPayload, String> {
    let providers_path = mobile_local_providers_path(&app)?;
    let secrets_path = mobile_local_provider_secrets_path(&app)?;
    let mut state = read_mobile_local_provider_state(&providers_path)?;
    let mut secrets = read_mobile_local_provider_secrets(&secrets_path)?;
    upsert_mobile_local_provider(&mut state, &mut secrets, provider)?;
    write_mobile_local_provider_state(&providers_path, &state)?;
    write_mobile_local_provider_secrets(&secrets_path, &secrets)?;
    Ok(build_mobile_local_provider_payload(state, &secrets))
}

#[tauri::command]
fn mobile_local_provider_delete(
    app: tauri::AppHandle,
    provider_id: String,
) -> Result<MobileLocalProviderListPayload, String> {
    let providers_path = mobile_local_providers_path(&app)?;
    let secrets_path = mobile_local_provider_secrets_path(&app)?;
    let mut state = read_mobile_local_provider_state(&providers_path)?;
    let secrets = read_mobile_local_provider_secrets(&secrets_path)?;
    let target = provider_id.trim();
    if target.is_empty() {
        return Err("provider_id is required".to_string());
    }
    let deleted_at = current_timestamp();
    for record in state.providers.iter_mut() {
        if record.provider_id == target {
            record.updated_at = deleted_at.clone();
            record.deleted_at = Some(deleted_at.clone());
        }
    }
    write_mobile_local_provider_state(&providers_path, &state)?;
    Ok(build_mobile_local_provider_payload(state, &secrets))
}

#[tauri::command]
fn mobile_local_provider_export(
    app: tauri::AppHandle,
    provider_id: String,
) -> Result<MobileLocalProviderExportPayload, String> {
    let providers_path = mobile_local_providers_path(&app)?;
    let secrets_path = mobile_local_provider_secrets_path(&app)?;
    let state = read_mobile_local_provider_state(&providers_path)?;
    let secrets = read_mobile_local_provider_secrets(&secrets_path)?;
    let target = provider_id.trim();
    if target.is_empty() {
        return Err("provider_id is required".to_string());
    }
    let provider = state
        .providers
        .into_iter()
        .find(|provider| provider.provider_id == target)
        .ok_or_else(|| format!("local provider not found: {target}"))?;
    Ok(MobileLocalProviderExportPayload {
        name: provider.name,
        provider_type: provider.provider_type,
        provider_id: provider.provider_id.clone(),
        updated_at: provider.updated_at,
        deleted_at: provider.deleted_at,
        base_url: provider.base_url,
        api_key: secrets.get(provider.provider_id.trim()).cloned(),
        models: provider.models,
        context_window_tokens: provider.context_window_tokens,
        response_reserve_tokens: provider.response_reserve_tokens,
        model_context_window_tokens: provider.model_context_window_tokens,
        model_response_reserve_tokens: provider.model_response_reserve_tokens,
    })
}

#[tauri::command]
async fn mobile_local_llm_send(
    app: tauri::AppHandle,
    request: MobileLocalLLMSendRequest,
) -> Result<MobileLocalLLMSendResponse, String> {
    let state = read_mobile_local_provider_state(&mobile_local_providers_path(&app)?)?;
    let secrets = read_mobile_local_provider_secrets(&mobile_local_provider_secrets_path(&app)?)?;
    let provider = state
        .providers
        .iter()
        .find(|provider| {
            provider.provider_id == request.provider_id.trim() && provider.deleted_at.is_none()
        })
        .cloned()
        .ok_or_else(|| format!("local provider not found: {}", request.provider_id.trim()))?;
    let api_key = secrets
        .get(request.provider_id.trim())
        .cloned()
        .ok_or_else(|| {
            format!(
                "local provider secret not found: {}",
                request.provider_id.trim()
            )
        })?;
    let message = send_mobile_local_llm_request(&provider, &api_key, &request).await?;
    Ok(MobileLocalLLMSendResponse {
        message,
        model: request.model.trim().to_string(),
        provider_id: request.provider_id.trim().to_string(),
    })
}

fn bridge_bus_url(base_url: &str) -> Result<reqwest::Url, String> {
    bridge_api_url(base_url, BRIDGE_BUS_PATH)
}

fn bridge_agent_stream_url(base_url: &str) -> Result<reqwest::Url, String> {
    bridge_stream_url(base_url, BRIDGE_AGENT_STREAM_PATH)
}

fn bridge_stream_url(base_url: &str, path: &str) -> Result<reqwest::Url, String> {
    bridge_api_url(base_url, path)
}

fn bridge_run_events_url(base_url: &str, trace_id: &str) -> Result<reqwest::Url, String> {
    let mut url = bridge_api_url(base_url, BRIDGE_RUN_EVENTS_PATH)?;
    url.path_segments_mut()
        .map_err(|_| "bridge URL cannot contain run event path segments".to_string())?
        .push(trace_id)
        .push("events");
    Ok(url)
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

async fn run_blocking_file_task<T, F>(label: &'static str, task: F) -> Result<T, String>
where
    T: Send + 'static,
    F: FnOnce() -> Result<T, String> + Send + 'static,
{
    tauri::async_runtime::spawn_blocking(task)
        .await
        .map_err(|err| format!("{label} task failed: {err}"))?
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

fn validate_agent_stream_request(
    request: &BridgeAgentStreamCommand,
) -> Result<&'static str, String> {
    if request.request_id.trim().is_empty() {
        return Err("stream request_id is required".to_string());
    }
    if request.trace_id.trim().is_empty() {
        return Err("trace_id is required".to_string());
    }
    let path = normalize_agent_stream_path(request.path.as_deref())?;
    if let Some(body) = request.body.as_ref() {
        validate_stream_body_message(body)?;
    } else if request.message.trim().is_empty() {
        return Err("agent message is required".to_string());
    }
    Ok(path)
}

fn normalize_agent_stream_path(path: Option<&str>) -> Result<&'static str, String> {
    match path.map(str::trim).filter(|value| !value.is_empty()) {
        None => Ok(BRIDGE_AGENT_STREAM_PATH),
        Some(BRIDGE_AGENT_STREAM_PATH) => Ok(BRIDGE_AGENT_STREAM_PATH),
        Some(BRIDGE_EXTERNAL_AGENT_STREAM_PATH) => Ok(BRIDGE_EXTERNAL_AGENT_STREAM_PATH),
        Some(other) => Err(format!("unsupported bridge stream path: {other}")),
    }
}

fn validate_stream_body_message(body: &Value) -> Result<(), String> {
    let message = body
        .get("message")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");
    if message.is_empty() {
        return Err("agent message is required".to_string());
    }
    Ok(())
}

fn build_agent_stream_body(request: &BridgeAgentStreamCommand) -> Result<Value, String> {
    if let Some(body) = request.body.clone() {
        return Ok(body);
    }

    let session_id = request
        .session_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty());
    serde_json::to_value(BridgeAgentStreamBody {
        message: request.message.trim(),
        runtime_overrides: request.runtime_overrides.as_ref(),
        session_id,
        trace_id: request.trace_id.trim(),
    })
    .map_err(|err| format!("encode bridge stream body failed: {err}"))
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
    let flush_interval = Duration::from_millis(STREAM_IPC_FLUSH_INTERVAL_MS);
    let flush_timer = tokio::time::sleep(flush_interval);
    pin_mut!(flush_timer);
    let mut pending = Vec::new();

    loop {
        if pending.is_empty() {
            let Some(item) = stream.next().await else {
                break;
            };
            let bytes = item.map_err(|err| format!("read bridge stream failed: {err}"))?;
            append_stream_bytes(&mut pending, &bytes);
            flush_timer
                .as_mut()
                .reset(tokio::time::Instant::now() + flush_interval);
            if pending.len() >= STREAM_IPC_MAX_BATCH_BYTES {
                emit_stream_chunk(&app, &request_id, std::mem::take(&mut pending))?;
            }
            continue;
        }

        let next_item = stream.next();
        pin_mut!(next_item);
        match select(next_item, flush_timer.as_mut()).await {
            Either::Left((item, _timer)) => {
                let Some(item) = item else {
                    break;
                };
                let bytes = item.map_err(|err| format!("read bridge stream failed: {err}"))?;
                append_stream_bytes(&mut pending, &bytes);
                if pending.len() >= STREAM_IPC_MAX_BATCH_BYTES {
                    emit_stream_chunk(&app, &request_id, std::mem::take(&mut pending))?;
                    flush_timer
                        .as_mut()
                        .reset(tokio::time::Instant::now() + flush_interval);
                }
            }
            Either::Right(((), _next_item)) => {
                emit_stream_chunk(&app, &request_id, std::mem::take(&mut pending))?;
            }
        }
    }
    emit_stream_chunk(&app, &request_id, pending)?;
    Ok(())
}

fn append_stream_bytes(pending: &mut Vec<u8>, bytes: &[u8]) {
    pending.extend_from_slice(bytes);
}

fn emit_stream_chunk(
    app: &tauri::AppHandle,
    request_id: &str,
    chunk: Vec<u8>,
) -> Result<(), String> {
    if chunk.is_empty() {
        return Ok(());
    }

    app.emit(
        BRIDGE_AGENT_STREAM_CHUNK_EVENT,
        BridgeAgentStreamChunk {
            request_id: request_id.to_string(),
            chunk,
        },
    )
    .map_err(|err| format!("emit bridge stream chunk failed: {err}"))
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

fn mobile_local_providers_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve local provider directory failed: {err}"))?;
    Ok(dir.join(MOBILE_LOCAL_PROVIDERS_FILE))
}

fn mobile_local_provider_secrets_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve local provider secrets directory failed: {err}"))?;
    Ok(dir.join(MOBILE_LOCAL_PROVIDER_SECRETS_FILE))
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

fn upsert_mobile_conversations(current: &mut Vec<Value>, incoming: Vec<Value>) {
    for conversation in incoming {
        let Some(id) = mobile_conversation_id(&conversation).map(str::to_string) else {
            continue;
        };
        match current
            .iter()
            .position(|item| mobile_conversation_id(item) == Some(id.as_str()))
        {
            Some(index) => current[index] = conversation,
            None => current.push(conversation),
        }
    }
}

fn compact_mobile_conversations(conversations: Vec<Value>) -> Vec<Value> {
    conversations
        .into_iter()
        .map(|mut conversation| {
            if let Value::Object(ref mut object) = conversation {
                object.insert("messages".to_string(), Value::Array(Vec::new()));
            }
            conversation
        })
        .collect()
}

fn sort_mobile_conversations(conversations: &mut [Value]) {
    conversations.sort_by(|left, right| {
        let updated_order =
            mobile_conversation_updated_at(right).cmp(mobile_conversation_updated_at(left));
        if !updated_order.is_eq() {
            return updated_order;
        }
        mobile_conversation_title(left).cmp(mobile_conversation_title(right))
    });
}

fn mobile_conversation_id(conversation: &Value) -> Option<&str> {
    conversation
        .as_object()
        .and_then(|object| object.get("id"))
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
}

fn mobile_conversation_updated_at(conversation: &Value) -> &str {
    conversation
        .as_object()
        .and_then(|object| object.get("updated_at"))
        .and_then(Value::as_str)
        .unwrap_or("")
}

fn mobile_conversation_title(conversation: &Value) -> &str {
    conversation
        .as_object()
        .and_then(|object| object.get("title"))
        .and_then(Value::as_str)
        .unwrap_or("")
}

fn read_mobile_local_provider_state(path: &Path) -> Result<MobileLocalProviderState, String> {
    match fs::read_to_string(path) {
        Ok(raw) => {
            if raw.trim().is_empty() {
                return Ok(MobileLocalProviderState::default());
            }
            serde_json::from_str(&raw)
                .map_err(|err| format!("decode local providers failed: {err}"))
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
            Ok(MobileLocalProviderState::default())
        }
        Err(err) => Err(format!("read local providers failed: {err}")),
    }
}

fn write_mobile_local_provider_state(
    path: &Path,
    state: &MobileLocalProviderState,
) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|err| format!("create local provider directory failed: {err}"))?;
    }
    let encoded = serde_json::to_vec_pretty(state)
        .map_err(|err| format!("encode local providers failed: {err}"))?;
    fs::write(path, encoded).map_err(|err| format!("write local providers failed: {err}"))?;
    set_private_file_permissions(path, "local providers")
}

fn read_mobile_local_provider_secrets(path: &Path) -> Result<BTreeMap<String, String>, String> {
    match fs::read_to_string(path) {
        Ok(raw) => {
            if raw.trim().is_empty() {
                return Ok(BTreeMap::new());
            }
            serde_json::from_str(&raw)
                .map_err(|err| format!("decode local provider secrets failed: {err}"))
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(BTreeMap::new()),
        Err(err) => Err(format!("read local provider secrets failed: {err}")),
    }
}

fn write_mobile_local_provider_secrets(
    path: &Path,
    secrets: &BTreeMap<String, String>,
) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|err| format!("create local provider secrets directory failed: {err}"))?;
    }
    let encoded = serde_json::to_vec_pretty(secrets)
        .map_err(|err| format!("encode local provider secrets failed: {err}"))?;
    fs::write(path, encoded)
        .map_err(|err| format!("write local provider secrets failed: {err}"))?;
    set_private_file_permissions(path, "local provider secrets")
}

fn build_mobile_local_provider_payload(
    state: MobileLocalProviderState,
    secrets: &BTreeMap<String, String>,
) -> MobileLocalProviderListPayload {
    let sync_records = state
        .providers
        .into_iter()
        .map(|record| mobile_local_provider_payload(record, secrets))
        .collect::<Vec<_>>();
    let providers = sync_records
        .iter()
        .filter(|record| record.deleted_at.is_none())
        .cloned()
        .collect::<Vec<_>>();
    MobileLocalProviderListPayload {
        providers,
        active_provider: String::new(),
        provider_sync_records: sync_records,
    }
}

fn mobile_local_provider_payload(
    record: MobileLocalProviderRecord,
    secrets: &BTreeMap<String, String>,
) -> MobileLocalProviderPayload {
    MobileLocalProviderPayload {
        api_key_set: secrets
            .get(record.provider_id.trim())
            .map(|secret| !secret.trim().is_empty())
            .unwrap_or(false),
        base_url: record.base_url,
        context_window_tokens: record.context_window_tokens,
        deleted_at: record.deleted_at,
        model_context_window_tokens: record.model_context_window_tokens,
        model_response_reserve_tokens: record.model_response_reserve_tokens,
        models: record.models,
        name: record.name,
        provider_id: record.provider_id,
        provider_type: record.provider_type,
        response_reserve_tokens: record.response_reserve_tokens,
        sync_state: "local",
        updated_at: record.updated_at,
    }
}

fn upsert_mobile_local_provider(
    state: &mut MobileLocalProviderState,
    secrets: &mut BTreeMap<String, String>,
    provider: MobileLocalProviderUpsertRequest,
) -> Result<(), String> {
    let name = provider.name.trim();
    if name.is_empty() {
        return Err("provider name is required".to_string());
    }
    let provider_id = provider
        .provider_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(str::to_string)
        .unwrap_or_else(new_local_provider_id);
    let updated_at = provider
        .updated_at
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(str::to_string)
        .unwrap_or_else(current_timestamp);
    let base_url = provider.base_url.unwrap_or_default().trim().to_string();
    let provider_type = provider.provider_type.trim().to_string();
    let mut next = MobileLocalProviderRecord {
        name: name.to_string(),
        provider_type,
        base_url,
        provider_id: provider_id.clone(),
        updated_at,
        deleted_at: provider.deleted_at.filter(|value| !value.trim().is_empty()),
        models: provider.models.unwrap_or_default(),
        context_window_tokens: provider.context_window_tokens,
        response_reserve_tokens: provider.response_reserve_tokens,
        model_context_window_tokens: provider.model_context_window_tokens,
        model_response_reserve_tokens: provider.model_response_reserve_tokens,
    };
    let mut replaced = false;
    for record in state.providers.iter_mut() {
        if record.provider_id == provider_id || record.name.trim().eq_ignore_ascii_case(name) {
            *record = next.clone();
            replaced = true;
            break;
        }
    }
    if !replaced {
        state.providers.insert(0, next);
    }
    if let Some(secret) = provider.api_key {
        let normalized = secret.trim().to_string();
        if normalized.is_empty() {
            secrets.remove(provider_id.trim());
        } else {
            secrets.insert(provider_id, normalized);
        }
    }
    Ok(())
}

fn new_local_provider_id() -> String {
    uuid::Uuid::new_v4().to_string()
}

fn current_timestamp() -> String {
    chrono::Utc::now().to_rfc3339()
}

async fn send_mobile_local_llm_request(
    provider: &MobileLocalProviderRecord,
    api_key: &str,
    request: &MobileLocalLLMSendRequest,
) -> Result<String, String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(REQUEST_TIMEOUT_SECS))
        .build()
        .map_err(|err| format!("create local LLM client failed: {err}"))?;
    let provider_type = provider.provider_type.trim().to_ascii_lowercase();
    if provider_type == "anthropic" {
        let url = bridge_api_url(&provider.base_url, "v1/messages")?;
        let messages = request
            .history
            .iter()
            .map(|message| {
                serde_json::json!({
                    "role": message.role,
                    "content": message.text,
                })
            })
            .collect::<Vec<_>>();
        let response = client
            .post(url)
            .header("x-api-key", api_key.trim())
            .header("anthropic-version", "2023-06-01")
            .json(&serde_json::json!({
                "max_tokens": 2048,
                "messages": messages,
                "model": request.model,
            }))
            .send()
            .await
            .map_err(|err| format!("local anthropic request failed: {err}"))?;
        let body = response
            .json::<Value>()
            .await
            .map_err(|err| format!("decode local anthropic response failed: {err}"))?;
        return Ok(body
            .get("content")
            .and_then(Value::as_array)
            .map(|items| {
                items
                    .iter()
                    .filter_map(|item| item.get("text").and_then(Value::as_str))
                    .collect::<Vec<_>>()
                    .join("")
            })
            .filter(|text| !text.trim().is_empty())
            .ok_or_else(|| "local anthropic response did not include text".to_string())?);
    }

    let url = bridge_api_url(&provider.base_url, "chat/completions")?;
    let messages = request
        .history
        .iter()
        .map(|message| {
            serde_json::json!({
                "role": message.role,
                "content": message.text,
            })
        })
        .collect::<Vec<_>>();
    let response = client
        .post(url)
        .header("Authorization", format!("Bearer {}", api_key.trim()))
        .json(&serde_json::json!({
            "messages": messages,
            "model": request.model,
        }))
        .send()
        .await
        .map_err(|err| format!("local LLM request failed: {err}"))?;
    let body = response
        .json::<Value>()
        .await
        .map_err(|err| format!("decode local LLM response failed: {err}"))?;
    body.get("choices")
        .and_then(Value::as_array)
        .and_then(|items| items.first())
        .and_then(|choice| choice.get("message"))
        .and_then(|message| message.get("content"))
        .and_then(Value::as_str)
        .map(str::to_string)
        .ok_or_else(|| "local LLM response did not include assistant content".to_string())
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
    fn bridge_run_events_url_encodes_trace_id() {
        let url =
            bridge_run_events_url("http://127.0.0.1:8711", "trace one").expect("run events URL");

        assert_eq!(
            url.as_str(),
            "http://127.0.0.1:8711/api/runs/trace%20one/events"
        );
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
            bridge_agent_stream_reconnect,
            bridge_bus_request,
            mobile_credential_save,
            mobile_credential_load,
            mobile_credential_delete,
            mobile_conversations_load,
            mobile_conversations_load_index,
            mobile_conversation_get,
            mobile_conversations_upsert,
            mobile_conversations_save,
            mobile_local_provider_list,
            mobile_local_provider_upsert,
            mobile_local_provider_delete,
            mobile_local_provider_export,
            mobile_local_llm_send
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
