use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeMap;
use std::fs;
use std::path::PathBuf;
use std::time::Duration;
use tauri::Manager;

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
        return Err(format!(
            "bridge returned HTTP {status} with success envelope"
        ));
    }
    Ok(decoded)
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
    let parsed =
        reqwest::Url::parse(&normalized).map_err(|err| format!("invalid bridge URL: {err}"))?;
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

fn mobile_credential_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_config_dir()
        .map_err(|err| format!("resolve mobile credential directory failed: {err}"))?;
    Ok(dir.join("mobile-credentials.json"))
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
    set_private_file_permissions(path)
}

fn set_private_file_permissions(path: &PathBuf) -> Result<(), String> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;

        fs::set_permissions(path, fs::Permissions::from_mode(0o600))
            .map_err(|err| format!("set mobile credential permissions failed: {err}"))?;
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

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            host_profile,
            bridge_bus_request,
            mobile_credential_save,
            mobile_credential_load,
            mobile_credential_delete
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
