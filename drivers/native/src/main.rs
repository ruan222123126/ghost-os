#![allow(unsafe_op_in_unsafe_fn)]

mod action_router;
mod browser_query;
mod sandbox;
mod script_exec;

use base64::Engine;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::fs;
use std::io::{self, Read, Write};
use std::process::Command;
use xcap::Monitor;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseButton, MouseControllable};

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
pub(crate) struct Response {
    status: String,
    payload: Value,
    error: String,
}

impl Response {
    // success 构造成功响应。
    pub(crate) fn success(payload: Value) -> Self {
        Self {
            status: "success".to_string(),
            payload,
            error: String::new(),
        }
    }

    // error 构造失败响应。
    pub(crate) fn error(message: String) -> Self {
        Self {
            status: "error".to_string(),
            payload: json!({}),
            error: message,
        }
    }
}

fn main() {
    if std::env::args().any(|arg| arg == "--sandbox-worker") {
        script_exec::run_sandbox_worker();
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

    let response =
        action_router::dispatch_action(&request.action, &request.params, &request.trace_id);
    emit(response);
}

pub(crate) fn read_stdin_payload() -> Result<String, String> {
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

pub(crate) fn handle_list_files(params: &Value) -> Response {
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

pub(crate) fn handle_bash_exec(params: &Value) -> Response {
    let command = params
        .get("command")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");

    if command.is_empty() {
        return Response::error("command is required".to_string());
    }

    // BASH_EXEC 入口尚未开放，避免返回 success 造成上层误判为“已执行”。
    Response::error(
        "BASH_EXEC is not implemented in native execution layer; use SCRIPT_EXEC tools.bash_exec"
            .to_string(),
    )
}

#[derive(Clone, Copy, Debug)]
enum ClickButton {
    Left,
    Right,
    Middle,
}

pub(crate) fn handle_screen_shot(params: &Value) -> Response {
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

pub(crate) fn handle_mouse_click(params: &Value) -> Response {
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

// emit 负责写出单行 JSON 响应，失败时静默返回。
fn emit(response: Response) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&response) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}

#[cfg(test)]
mod tests {
    use super::{button_name, handle_bash_exec, parse_click_button};
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
    fn handle_bash_exec_returns_error_when_command_is_missing() {
        let response = handle_bash_exec(&json!({}));
        assert_eq!(response.status, "error");
        assert_eq!(response.error, "command is required");
    }

    #[test]
    fn handle_bash_exec_returns_error_when_unimplemented() {
        let response = handle_bash_exec(&json!({"command":"echo hi"}));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("not implemented"));
    }
}
