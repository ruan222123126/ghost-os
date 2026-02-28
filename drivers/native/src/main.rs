#![allow(unsafe_op_in_unsafe_fn)]

mod sandbox;

use base64::Engine;
use reqwest::blocking::Client as HttpClient;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::fs;
use std::io::{self, BufReader, Read, Write};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant};
use xcap::Monitor;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseButton, MouseControllable};

use sandbox::{ExecutionResult, PythonSandbox, SandboxConfig};

// Request 对齐 core/shared/schema.json 的请求结构。
#[derive(Deserialize)]
struct Request {
    action: String,
    #[serde(rename = "params", default)]
    params: Value,
    #[serde(rename = "trace_id", default)]
    trace_id: String,
}

// Response 是 native 层统一返回格式。
#[derive(Serialize)]
struct Response {
    status: String,
    payload: Value,
    error: String,
}

impl Response {
    // success 构造成功响应。
    fn success(payload: Value) -> Self {
        Self {
            status: "success".to_string(),
            payload,
            error: String::new(),
        }
    }

    // error 构造失败响应。
    fn error(message: String) -> Self {
        Self {
            status: "error".to_string(),
            payload: json!({}),
            error: message,
        }
    }
}

#[derive(Deserialize, Serialize)]
struct ScriptWorkerRequest {
    script: String,
    max_memory_mb: u64,
}

fn main() {
    if std::env::args().any(|arg| arg == "--sandbox-worker") {
        run_sandbox_worker();
        return;
    }

    // 读取整段 stdin，保持最小协议处理路径。
    let input = match read_stdin_payload() {
        Ok(input) => input,
        Err(err) => {
            emit(Response::error(err));
            return;
        }
    };

    let request: Request = match serde_json::from_str(&input) {
        Ok(request) => request,
        Err(err) => {
            emit(Response::error(format!("invalid json: {err}")));
            return;
        }
    };

    let response = match request.action.as_str() {
        "PING" => Response::success(json!({
            "message": "PONG",
            "trace_id": request.trace_id
        })),
        "LIST_FILES" => handle_list_files(&request.params),
        "BASH_EXEC" => handle_bash_exec(&request.params),
        "SCRIPT_EXEC" => handle_script_exec(&request.params),
        "SCREEN_SHOT" => handle_screen_shot(&request.params),
        "MOUSE_CLICK" => handle_mouse_click(&request.params),
        "BROWSER_QUERY" => handle_browser_query(&request.params),
        _ => Response::error(format!("unsupported action: {}", request.action)),
    };

    emit(response);
}

fn run_sandbox_worker() {
    let input = match read_stdin_payload() {
        Ok(input) => input,
        Err(err) => {
            emit_worker_result(ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(err),
            });
            return;
        }
    };

    let request: ScriptWorkerRequest = match serde_json::from_str(&input) {
        Ok(request) => request,
        Err(err) => {
            emit_worker_result(ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(format!("invalid sandbox worker request: {err}")),
            });
            return;
        }
    };

    if let Err(err) = apply_memory_limit(request.max_memory_mb) {
        emit_worker_result(ExecutionResult {
            output: String::new(),
            tool_calls_log: Vec::new(),
            error: Some(err),
        });
        return;
    }

    let mut config = SandboxConfig::default();
    config.max_memory_mb = request.max_memory_mb;
    let sandbox = PythonSandbox::new(config);
    let result = sandbox.execute_blocking(&request.script);
    emit_worker_result(result);
}

fn read_stdin_payload() -> Result<String, String> {
    let mut input = String::new();
    io::stdin()
        .read_to_string(&mut input)
        .map_err(|_| "failed to read stdin".to_string())?;

    let input = input.trim();
    if input.is_empty() {
        return Err("empty input".to_string());
    }

    Ok(input.to_string())
}

fn handle_list_files(params: &Value) -> Response {
    let path = params
        .get("path")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".");

    let read_dir = match fs::read_dir(path) {
        Ok(entries) => entries,
        Err(err) => {
            return Response::error(format!("read dir {path:?} failed: {err}"));
        }
    };

    let mut names = Vec::new();
    for entry_result in read_dir {
        let entry = match entry_result {
            Ok(entry) => entry,
            Err(err) => return Response::error(format!("read dir entry failed: {err}")),
        };

        let metadata = match entry.metadata() {
            Ok(metadata) => metadata,
            Err(err) => return Response::error(format!("read metadata failed: {err}")),
        };

        let mut name = entry.file_name().to_string_lossy().to_string();
        if metadata.is_dir() {
            name.push('/');
        }
        names.push(name);
    }

    names.sort();
    Response::success(json!({
        "path": path,
        "entries": names,
    }))
}

fn handle_bash_exec(params: &Value) -> Response {
    let command = params
        .get("command")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");

    if command.is_empty() {
        return Response::error("command is required".to_string());
    }

    // 实际 shell 执行策略尚未开放，这里保留 execution layer 受控扩展点。
    Response::success(json!({
        "output": format!("[stub] execution layer received command: {command}"),
    }))
}

fn handle_script_exec(params: &Value) -> Response {
    let script = params
        .get("script")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");

    if script.is_empty() {
        return Response::error("script is required".to_string());
    }

    let mut timeout_ms = params
        .get("timeout_ms")
        .and_then(Value::as_u64)
        .unwrap_or(30_000);
    if timeout_ms == 0 {
        timeout_ms = 30_000;
    }
    if timeout_ms > 60_000 {
        timeout_ms = 60_000;
    }

    let mut max_memory_mb = params
        .get("max_memory_mb")
        .and_then(Value::as_u64)
        .unwrap_or(256);
    if max_memory_mb == 0 {
        max_memory_mb = 256;
    }
    if max_memory_mb > 512 {
        max_memory_mb = 512;
    }

    let result = match execute_script_in_subprocess(script, timeout_ms, max_memory_mb) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    if let Some(err) = result.error {
        return Response::error(err);
    }

    Response::success(json!({
        "output": result.output,
        "tool_calls_log": result.tool_calls_log,
    }))
}

#[derive(Clone, Copy, Debug)]
enum ClickButton {
    Left,
    Right,
    Middle,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BrowserQueryKind {
    ActiveTab,
    WindowList,
    Url,
}

#[derive(Debug)]
struct BrowserQueryOptions {
    query_type: BrowserQueryKind,
    debug_host: String,
    debug_port: Option<u16>,
    include_non_browser: bool,
}

#[derive(Clone, Debug, Serialize)]
struct BrowserWindowInfo {
    window_id: String,
    title: String,
    class_name: String,
    pid: Option<i32>,
    is_active: bool,
    is_browser: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct CdpTargetInfo {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    url: String,
    #[serde(default, rename = "type")]
    target_type: String,
    #[serde(skip)]
    debug_port: u16,
}

fn handle_screen_shot(params: &Value) -> Response {
    let display_id = match parse_optional_display_id(params) {
        Ok(display_id) => display_id,
        Err(err) => return Response::error(err),
    };

    let monitors = match Monitor::all() {
        Ok(monitors) => monitors,
        Err(err) => return Response::error(format!("list monitors failed: {err}")),
    };
    if monitors.is_empty() {
        return Response::error("no monitor is available".to_string());
    }

    let monitor = if let Some(id) = display_id {
        match monitors.iter().find(|monitor| monitor.id() == id) {
            Some(monitor) => monitor,
            None => {
                return Response::error(format!(
                    "display_id {id} is not found; available displays: {}",
                    monitors
                        .iter()
                        .map(|monitor| monitor.id().to_string())
                        .collect::<Vec<_>>()
                        .join(",")
                ));
            }
        }
    } else {
        monitors
            .iter()
            .find(|monitor| monitor.is_primary())
            .unwrap_or(&monitors[0])
    };

    let image = match monitor.capture_image() {
        Ok(image) => image,
        Err(err) => {
            return Response::error(format!("capture monitor {} failed: {err}", monitor.id()));
        }
    };

    let width = image.width();
    let height = image.height();
    let png_bytes = match encode_png(image) {
        Ok(png_bytes) => png_bytes,
        Err(err) => return Response::error(err),
    };
    let image_base64 = base64::engine::general_purpose::STANDARD.encode(png_bytes);

    Response::success(json!({
        "image_base64": image_base64,
        "width": width,
        "height": height,
        "display_id": monitor.id(),
    }))
}

fn handle_mouse_click(params: &Value) -> Response {
    let x = match parse_required_i32(params, "x") {
        Ok(x) => x,
        Err(err) => return Response::error(err),
    };
    let y = match parse_required_i32(params, "y") {
        Ok(y) => y,
        Err(err) => return Response::error(err),
    };
    let button = match parse_click_button(params) {
        Ok(button) => button,
        Err(err) => return Response::error(err),
    };

    if let Err(err) = perform_mouse_click(x, y, button) {
        return Response::error(err);
    }

    Response::success(json!({
        "clicked": true,
        "x": x,
        "y": y,
        "button": button_name(button),
    }))
}

fn handle_browser_query(params: &Value) -> Response {
    let options = match parse_browser_query_options(params) {
        Ok(options) => options,
        Err(err) => return Response::error(err),
    };

    let mut windows_error: Option<String> = None;
    let windows = match query_windows(options.include_non_browser) {
        Ok(windows) => windows,
        Err(err) => {
            windows_error = Some(err);
            Vec::new()
        }
    };

    let mut cdp_error: Option<String> = None;
    let cdp_targets = match query_cdp_targets(&options.debug_host, options.debug_port) {
        Ok(targets) => targets,
        Err(err) => {
            cdp_error = Some(err);
            Vec::new()
        }
    };

    match options.query_type {
        BrowserQueryKind::WindowList => {
            if !windows.is_empty() {
                return Response::success(json!({
                    "status": "ok",
                    "query_type": "window_list",
                    "count": windows.len(),
                    "windows": windows,
                }));
            }
            if !cdp_targets.is_empty() {
                return Response::success(json!({
                    "status": "ok",
                    "query_type": "window_list",
                    "count": 0,
                    "windows": [],
                    "tab_count": cdp_targets.len(),
                    "tabs": cdp_targets,
                    "source": "cdp",
                }));
            }
            Response::error(compose_browser_query_error(
                "window_list query failed",
                windows_error,
                cdp_error,
            ))
        }
        BrowserQueryKind::ActiveTab => {
            let active_window = pick_active_window(&windows);
            let matched_tab = active_window
                .as_ref()
                .and_then(|window| match_target_for_window(window, &cdp_targets))
                .or_else(|| cdp_targets.first());

            if active_window.is_none() && matched_tab.is_none() {
                return Response::error(compose_browser_query_error(
                    "active_tab query failed",
                    windows_error,
                    cdp_error,
                ));
            }

            let source = match (active_window.is_some(), matched_tab.is_some()) {
                (true, true) => "window+cdp",
                (true, false) => "window",
                (false, true) => "cdp",
                (false, false) => "unknown",
            };

            Response::success(json!({
                "status": "ok",
                "query_type": "active_tab",
                "window": active_window,
                "tab": matched_tab,
                "source": source,
            }))
        }
        BrowserQueryKind::Url => {
            let tab = pick_active_window(&windows)
                .and_then(|window| match_target_for_window(&window, &cdp_targets))
                .or_else(|| cdp_targets.first());

            let Some(tab) = tab else {
                return Response::error(compose_browser_query_error(
                    "url query failed: no CDP tab is available",
                    windows_error,
                    cdp_error,
                ));
            };

            let url = tab.url.trim();
            if url.is_empty() {
                return Response::error("url query failed: active tab URL is empty".to_string());
            }

            Response::success(json!({
                "status": "ok",
                "query_type": "url",
                "url": url,
                "title": tab.title,
                "tab_id": tab.id,
                "debug_port": tab.debug_port,
                "source": "cdp",
            }))
        }
    }
}

fn parse_browser_query_options(params: &Value) -> Result<BrowserQueryOptions, String> {
    let query_type_raw = params
        .get("query_type")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("active_tab")
        .to_ascii_lowercase();

    let query_type = match query_type_raw.as_str() {
        "active_tab" => BrowserQueryKind::ActiveTab,
        "window_list" => BrowserQueryKind::WindowList,
        "url" => BrowserQueryKind::Url,
        _ => {
            return Err(format!(
                "query_type must be one of: active_tab, window_list, url; got {query_type_raw}"
            ));
        }
    };

    let debug_host = params
        .get("debug_host")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("127.0.0.1")
        .to_string();

    let debug_port = parse_optional_u16(params, "debug_port")?;
    let include_non_browser = params
        .get("include_non_browser")
        .and_then(Value::as_bool)
        .unwrap_or(false);

    Ok(BrowserQueryOptions {
        query_type,
        debug_host,
        debug_port,
        include_non_browser,
    })
}

fn parse_optional_u16(params: &Value, field: &str) -> Result<Option<u16>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    let number = raw
        .as_u64()
        .ok_or_else(|| format!("{field} must be a non-negative integer"))?;
    let value = u16::try_from(number).map_err(|_| format!("{field} must be <= 65535"))?;
    if value == 0 {
        return Err(format!("{field} must be >= 1"));
    }
    Ok(Some(value))
}

fn compose_browser_query_error(
    prefix: &str,
    windows_error: Option<String>,
    cdp_error: Option<String>,
) -> String {
    match (windows_error, cdp_error) {
        (Some(windows), Some(cdp)) => format!("{prefix}: {windows}; {cdp}"),
        (Some(windows), None) => format!("{prefix}: {windows}"),
        (None, Some(cdp)) => format!("{prefix}: {cdp}"),
        (None, None) => prefix.to_string(),
    }
}

fn pick_active_window(windows: &[BrowserWindowInfo]) -> Option<BrowserWindowInfo> {
    windows
        .iter()
        .find(|window| window.is_active)
        .cloned()
        .or_else(|| windows.first().cloned())
}

fn match_target_for_window<'a>(
    window: &BrowserWindowInfo,
    targets: &'a [CdpTargetInfo],
) -> Option<&'a CdpTargetInfo> {
    let normalized_window_title = normalize_window_title(&window.title);
    if !normalized_window_title.is_empty() {
        for target in targets {
            let normalized_target_title = normalize_window_title(&target.title);
            if normalized_target_title.is_empty() {
                continue;
            }
            if normalized_window_title == normalized_target_title
                || normalized_window_title.contains(&normalized_target_title)
                || normalized_target_title.contains(&normalized_window_title)
            {
                return Some(target);
            }
        }
    }

    targets.iter().find(|target| !target.url.trim().is_empty())
}

fn normalize_window_title(raw: &str) -> String {
    let mut value = raw.trim().to_ascii_lowercase();
    for suffix in [
        " - google chrome",
        " - chromium",
        " - mozilla firefox",
        " - brave",
        " - microsoft edge",
        " - opera",
        " - vivaldi",
        " - safari",
        " - arc",
    ] {
        if value.ends_with(suffix) {
            value.truncate(value.len().saturating_sub(suffix.len()));
            break;
        }
    }

    value
        .split_whitespace()
        .filter(|part| !part.is_empty())
        .collect::<Vec<_>>()
        .join(" ")
}

fn query_cdp_targets(
    debug_host: &str,
    debug_port: Option<u16>,
) -> Result<Vec<CdpTargetInfo>, String> {
    const DEFAULT_PORTS: [u16; 4] = [9222, 9223, 9229, 9333];

    let ports = if let Some(port) = debug_port {
        vec![port]
    } else {
        DEFAULT_PORTS.to_vec()
    };

    let client = HttpClient::builder()
        .timeout(Duration::from_millis(450))
        .build()
        .map_err(|err| format!("create HTTP client failed: {err}"))?;

    let mut errors = Vec::new();
    for port in ports {
        let endpoint = format!("http://{debug_host}:{port}/json/list");
        let response = match client.get(&endpoint).send() {
            Ok(response) => response,
            Err(err) => {
                errors.push(format!("{endpoint}: {err}"));
                continue;
            }
        };

        if !response.status().is_success() {
            errors.push(format!("{endpoint}: HTTP {}", response.status()));
            continue;
        }

        let body = match response.text() {
            Ok(body) => body,
            Err(err) => {
                errors.push(format!("{endpoint}: read body failed: {err}"));
                continue;
            }
        };
        let mut targets: Vec<CdpTargetInfo> = match serde_json::from_str(&body) {
            Ok(targets) => targets,
            Err(err) => {
                errors.push(format!("{endpoint}: decode json failed: {err}"));
                continue;
            }
        };

        targets.retain(|target| {
            target.target_type.eq_ignore_ascii_case("page")
                && !target.url.starts_with("devtools://")
                && !target.url.starts_with("chrome://")
        });

        if targets.is_empty() {
            errors.push(format!("{endpoint}: no page target"));
            continue;
        }

        for target in &mut targets {
            target.debug_port = port;
        }
        return Ok(targets);
    }

    if errors.is_empty() {
        return Err("CDP target discovery failed".to_string());
    }
    Err(format!(
        "CDP target discovery failed; ensure browser remote debugging is enabled (example: --remote-debugging-port=9222). details: {}",
        errors.join(" | ")
    ))
}

#[cfg(target_os = "linux")]
fn query_windows(include_non_browser: bool) -> Result<Vec<BrowserWindowInfo>, String> {
    let active_window_id = query_active_window_id_linux().ok();
    let output = Command::new("wmctrl")
        .args(["-lxp"])
        .output()
        .map_err(|err| format!("spawn wmctrl failed: {err}"))?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let detail = stderr.lines().next().unwrap_or("").trim();
        if detail.is_empty() {
            return Err("wmctrl returned non-zero status".to_string());
        }
        return Err(format!("wmctrl failed: {detail}"));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut windows = Vec::new();
    for line in stdout.lines() {
        let mut parts = line.split_whitespace();
        let Some(window_id_raw) = parts.next() else {
            continue;
        };
        let _desktop = parts.next();
        let pid_raw = parts.next();
        let _host = parts.next();
        let Some(class_name_raw) = parts.next() else {
            continue;
        };
        let title = parts.collect::<Vec<_>>().join(" ");

        let class_name = class_name_raw.to_string();
        let is_browser = is_browser_window_class(&class_name);
        if !include_non_browser && !is_browser {
            continue;
        }

        let window_id_num = parse_window_id(window_id_raw).unwrap_or(0);
        let is_active = active_window_id
            .map(|active_id| active_id == window_id_num)
            .unwrap_or(false);
        windows.push(BrowserWindowInfo {
            window_id: window_id_raw.to_string(),
            title,
            class_name,
            pid: pid_raw.and_then(|value| value.parse::<i32>().ok()),
            is_active,
            is_browser,
        });
    }

    Ok(windows)
}

#[cfg(not(target_os = "linux"))]
fn query_windows(_include_non_browser: bool) -> Result<Vec<BrowserWindowInfo>, String> {
    Err("window_list query is currently supported on Linux only".to_string())
}

#[cfg(target_os = "linux")]
fn query_active_window_id_linux() -> Result<u64, String> {
    let output = command_stdout(Command::new("xdotool").arg("getactivewindow"))?;
    output
        .trim()
        .parse::<u64>()
        .map_err(|err| format!("parse active window id failed: {err}"))
}

#[cfg(target_os = "linux")]
fn command_stdout(command: &mut Command) -> Result<String, String> {
    let output = command
        .output()
        .map_err(|err| format!("spawn command failed: {err}"))?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let detail = stderr.lines().next().unwrap_or("").trim();
        if detail.is_empty() {
            return Err(format!("command exited with status {}", output.status));
        }
        return Err(detail.to_string());
    }

    let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
    if stdout.is_empty() {
        return Err("command returned empty output".to_string());
    }
    Ok(stdout)
}

fn parse_window_id(raw: &str) -> Option<u64> {
    let value = raw.trim();
    if value.is_empty() {
        return None;
    }
    if let Some(hex) = value
        .strip_prefix("0x")
        .or_else(|| value.strip_prefix("0X"))
    {
        return u64::from_str_radix(hex, 16).ok();
    }
    value.parse::<u64>().ok()
}

fn is_browser_window_class(class_name: &str) -> bool {
    let normalized = class_name.to_ascii_lowercase();
    [
        "firefox", "chrome", "chromium", "brave", "edge", "opera", "vivaldi", "safari", "arc",
    ]
    .iter()
    .any(|keyword| normalized.contains(keyword))
}

fn parse_optional_display_id(params: &Value) -> Result<Option<u32>, String> {
    let Some(raw) = params.get("display_id") else {
        return Ok(None);
    };

    let value = raw
        .as_u64()
        .ok_or_else(|| "display_id must be a non-negative integer".to_string())?;
    let display_id = u32::try_from(value).map_err(|_| "display_id is too large".to_string())?;
    Ok(Some(display_id))
}

fn parse_required_i32(params: &Value, field: &str) -> Result<i32, String> {
    let raw = params
        .get(field)
        .ok_or_else(|| format!("{field} is required"))?;
    let value = raw
        .as_i64()
        .ok_or_else(|| format!("{field} must be an integer"))?;
    i32::try_from(value).map_err(|_| format!("{field} is out of i32 range"))
}

fn parse_click_button(params: &Value) -> Result<ClickButton, String> {
    let raw = params
        .get("button")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("left");

    match raw.to_ascii_lowercase().as_str() {
        "left" => Ok(ClickButton::Left),
        "right" => Ok(ClickButton::Right),
        "middle" => Ok(ClickButton::Middle),
        _ => Err("button must be one of: left, right, middle".to_string()),
    }
}

fn button_name(button: ClickButton) -> &'static str {
    match button {
        ClickButton::Left => "left",
        ClickButton::Right => "right",
        ClickButton::Middle => "middle",
    }
}

fn encode_png(image: xcap::image::RgbaImage) -> Result<Vec<u8>, String> {
    let mut cursor = std::io::Cursor::new(Vec::new());
    xcap::image::DynamicImage::ImageRgba8(image)
        .write_to(&mut cursor, xcap::image::ImageFormat::Png)
        .map_err(|err| format!("encode png failed: {err}"))?;
    Ok(cursor.into_inner())
}

#[cfg(target_os = "linux")]
fn perform_mouse_click(x: i32, y: i32, button: ClickButton) -> Result<(), String> {
    let button_id = match button {
        ClickButton::Left => "1",
        ClickButton::Middle => "2",
        ClickButton::Right => "3",
    };

    let status = Command::new("xdotool")
        .args([
            "mousemove",
            "--sync",
            &x.to_string(),
            &y.to_string(),
            "click",
            button_id,
        ])
        .status()
        .map_err(|err| format!("spawn xdotool failed: {err}"))?;

    if !status.success() {
        return Err(format!(
            "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
        ));
    }
    Ok(())
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_mouse_click(x: i32, y: i32, button: ClickButton) -> Result<(), String> {
    let mut enigo = Enigo::new();
    enigo.mouse_move_to(x, y);

    let native_button = match button {
        ClickButton::Left => MouseButton::Left,
        ClickButton::Right => MouseButton::Right,
        ClickButton::Middle => MouseButton::Middle,
    };
    enigo.mouse_click(native_button);
    Ok(())
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_mouse_click(_x: i32, _y: i32, _button: ClickButton) -> Result<(), String> {
    Err("mouse click is not supported on this platform".to_string())
}

fn execute_script_in_subprocess(
    script: &str,
    timeout_ms: u64,
    max_memory_mb: u64,
) -> Result<ExecutionResult, String> {
    let worker_request = ScriptWorkerRequest {
        script: script.to_string(),
        max_memory_mb,
    };

    let mut child = Command::new(
        std::env::current_exe()
            .map_err(|err| format!("resolve current executable failed: {err}"))?,
    )
    .arg("--sandbox-worker")
    .stdin(Stdio::piped())
    .stdout(Stdio::piped())
    .stderr(Stdio::piped())
    .spawn()
    .map_err(|err| format!("spawn sandbox worker failed: {err}"))?;

    let mut stdin = child
        .stdin
        .take()
        .ok_or_else(|| "sandbox worker stdin is not available".to_string())?;
    if let Err(err) = serde_json::to_writer(&mut stdin, &worker_request) {
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("encode sandbox worker request failed: {err}"));
    }
    if let Err(err) = stdin.write_all(b"\n") {
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("flush sandbox worker request failed: {err}"));
    }
    drop(stdin);

    let stdout = match child.stdout.take() {
        Some(stdout) => stdout,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err("sandbox worker stdout is not available".to_string());
        }
    };
    let stderr = match child.stderr.take() {
        Some(stderr) => stderr,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err("sandbox worker stderr is not available".to_string());
        }
    };

    let stdout_handle = spawn_pipe_reader(stdout);
    let stderr_handle = spawn_pipe_reader(stderr);

    let status = match wait_child_with_timeout(&mut child, Duration::from_millis(timeout_ms)) {
        Ok(Some(status)) => status,
        Ok(None) => {
            let _ = stdout_handle.join();
            let _ = stderr_handle.join();
            return Err(format!("script execution timeout after {}ms", timeout_ms));
        }
        Err(err) => {
            let _ = stdout_handle.join();
            let _ = stderr_handle.join();
            return Err(err);
        }
    };

    let stdout_bytes = stdout_handle.join().unwrap_or_default();
    let stderr_bytes = stderr_handle.join().unwrap_or_default();

    if !status.success() {
        let stderr_text = String::from_utf8_lossy(&stderr_bytes);
        let stderr_line = stderr_text.lines().next().unwrap_or("").trim();
        if stderr_line.is_empty() {
            return Err(format!("sandbox worker exited with status {}", status));
        }
        return Err(format!(
            "sandbox worker exited with status {}: {}",
            status, stderr_line
        ));
    }

    serde_json::from_slice::<ExecutionResult>(&stdout_bytes).map_err(|err| {
        let stdout_text = String::from_utf8_lossy(&stdout_bytes);
        let stdout_line = stdout_text.lines().next().unwrap_or("").trim();
        if stdout_line.is_empty() {
            format!("decode sandbox worker response failed: {err}")
        } else {
            format!(
                "decode sandbox worker response failed: {} (stdout: {})",
                err, stdout_line
            )
        }
    })
}

fn spawn_pipe_reader<R>(reader: R) -> thread::JoinHandle<Vec<u8>>
where
    R: Read + Send + 'static,
{
    thread::spawn(move || {
        let mut buffer = Vec::new();
        let mut reader = BufReader::new(reader);
        let _ = reader.read_to_end(&mut buffer);
        buffer
    })
}

fn wait_child_with_timeout(
    child: &mut Child,
    timeout: Duration,
) -> Result<Option<ExitStatus>, String> {
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => return Ok(Some(status)),
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Ok(None);
                }
                thread::sleep(Duration::from_millis(10));
            }
            Err(err) => return Err(format!("wait sandbox worker failed: {err}")),
        }
    }
}

#[cfg(unix)]
fn apply_memory_limit(max_memory_mb: u64) -> Result<(), String> {
    let memory_bytes = max_memory_mb
        .checked_mul(1024)
        .and_then(|value| value.checked_mul(1024))
        .ok_or_else(|| "max_memory_mb is too large".to_string())?;

    if (memory_bytes as u128) > (libc::rlim_t::MAX as u128) {
        return Err("max_memory_mb exceeds platform limit".to_string());
    }

    let rlim = memory_bytes as libc::rlim_t;
    let limit = libc::rlimit {
        rlim_cur: rlim,
        rlim_max: rlim,
    };

    // 使用进程级地址空间限制，确保沙盒超限时被系统拒绝继续分配内存。
    let rc = unsafe { libc::setrlimit(libc::RLIMIT_AS, &limit) };
    if rc != 0 {
        return Err(format!(
            "failed to apply memory limit: {}",
            io::Error::last_os_error()
        ));
    }

    Ok(())
}

#[cfg(not(unix))]
fn apply_memory_limit(_max_memory_mb: u64) -> Result<(), String> {
    Ok(())
}

// emit 负责写出单行 JSON 响应，失败时静默返回。
fn emit(response: Response) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&response) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}

fn emit_worker_result(result: ExecutionResult) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&result) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}

#[cfg(test)]
mod tests {
    use super::{
        BrowserQueryKind, BrowserWindowInfo, CdpTargetInfo, button_name, is_browser_window_class,
        match_target_for_window, normalize_window_title, parse_browser_query_options,
        parse_click_button, parse_window_id,
    };
    use serde_json::json;

    #[test]
    fn parse_click_button_defaults_to_left() {
        let button = parse_click_button(&json!({})).expect("button should parse");
        assert_eq!(button_name(button), "left");
    }

    #[test]
    fn parse_click_button_rejects_unsupported_values() {
        let err = parse_click_button(&json!({"button":"forward"})).expect_err("must fail");
        assert!(err.contains("button must be one of"));
    }

    #[test]
    fn parse_browser_query_options_defaults_to_active_tab() {
        let options = parse_browser_query_options(&json!({})).expect("must parse");
        assert_eq!(options.query_type, BrowserQueryKind::ActiveTab);
        assert_eq!(options.debug_host, "127.0.0.1");
        assert_eq!(options.debug_port, None);
        assert!(!options.include_non_browser);
    }

    #[test]
    fn parse_browser_query_options_rejects_unknown_query_type() {
        let err =
            parse_browser_query_options(&json!({"query_type":"unknown"})).expect_err("must fail");
        assert!(err.contains("query_type must be one of"));
    }

    #[test]
    fn parse_window_id_supports_hex_and_decimal() {
        assert_eq!(parse_window_id("0x10"), Some(16));
        assert_eq!(parse_window_id("15"), Some(15));
        assert_eq!(parse_window_id(""), None);
    }

    #[test]
    fn normalize_window_title_strips_browser_suffix() {
        assert_eq!(
            normalize_window_title("Ghost-OS - Google Chrome"),
            "ghost-os"
        );
        assert_eq!(normalize_window_title("  Example   Tab   "), "example tab");
    }

    #[test]
    fn is_browser_window_class_matches_known_browsers() {
        assert!(is_browser_window_class("Navigator.Firefox"));
        assert!(is_browser_window_class("google-chrome.Google-chrome"));
        assert!(!is_browser_window_class("code.Code"));
    }

    #[test]
    fn match_target_for_window_uses_title_similarity() {
        let window = BrowserWindowInfo {
            window_id: "0x1".to_string(),
            title: "Ghost-OS Dashboard - Google Chrome".to_string(),
            class_name: "google-chrome.Google-chrome".to_string(),
            pid: Some(1),
            is_active: true,
            is_browser: true,
        };
        let targets = vec![
            CdpTargetInfo {
                id: "tab-a".to_string(),
                title: "Other".to_string(),
                url: "https://example.com".to_string(),
                target_type: "page".to_string(),
                debug_port: 9222,
            },
            CdpTargetInfo {
                id: "tab-b".to_string(),
                title: "Ghost-OS Dashboard".to_string(),
                url: "https://ghost.local/dashboard".to_string(),
                target_type: "page".to_string(),
                debug_port: 9222,
            },
        ];

        let matched = match_target_for_window(&window, &targets).expect("must match");
        assert_eq!(matched.id, "tab-b");
    }
}
