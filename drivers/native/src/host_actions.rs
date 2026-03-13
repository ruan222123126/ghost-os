use crate::Response;
use crate::display_scale::{lookup_display_scale, record_display_scale};
use crate::sandbox::SandboxConfig;
use crate::sandbox::shell_tools::{
    bash_exec_impl, format_bash_exec_failure, resolve_shell_timeout,
};
use base64::Engine;
use serde_json::{Value, json};
use std::process::Command;
use xcap::Monitor;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, Key, KeyboardControllable, MouseButton, MouseControllable};

#[cfg(target_os = "linux")]
use std::env;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "BASH_EXEC" => Some(handle_bash_exec(params)),
        "SCREEN_SHOT" => Some(handle_screen_shot(params)),
        "TEXT_INPUT" => Some(handle_text_input(params)),
        "MOUSE_CLICK" => Some(handle_mouse_click(params)),
        _ => None,
    }
}

pub(crate) fn handle_bash_exec(params: &Value) -> Response {
    let command = match parse_required_string(params, "command") {
        Ok(command) => command,
        Err(err) => return Response::error(err),
    };

    let config = SandboxConfig::default();
    let timeout_ms = match parse_optional_usize(params, "timeout_ms") {
        Ok(timeout_ms) => resolve_shell_timeout(&config, timeout_ms.map(|value| value as u64)),
        Err(err) => return Response::error(err),
    };

    let result = match bash_exec_impl(&config, &command, Some(timeout_ms)) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };
    if !result.success {
        return Response::error(format_bash_exec_failure(&result));
    }

    Response::success(json!({
        "stdout": result.stdout,
        "stderr": result.stderr,
    }))
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
    let (scale_x, scale_y) = monitor_scale(&monitor, width, height);
    record_display_scale(monitor.id(), scale_x, scale_y);
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
        "scale_x": scale_x,
        "scale_y": scale_y,
    }))
}

pub(crate) fn handle_text_input(params: &Value) -> Response {
    let text = match parse_required_input_text(params, "text") {
        Ok(text) => text,
        Err(err) => return Response::error(err),
    };
    let submit = match parse_optional_bool(params, "submit") {
        Ok(submit) => submit,
        Err(err) => return Response::error(err),
    };
    let normalized = normalize_input_text(&text);

    if let Err(err) = perform_text_input(&normalized, submit) {
        return Response::error(err);
    }

    Response::success(json!({
        "typed": true,
        "submitted": submit,
        "characters": normalized.chars().count(),
        "lines": normalized.split('\n').count(),
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
    let display_id = match parse_optional_display_id(params) {
        Ok(display_id) => display_id,
        Err(err) => return Response::error(err),
    };
    let scale_x = match parse_optional_f64(params, "scale_x") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let scale_y = match parse_optional_f64(params, "scale_y") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let ensure_title = match parse_optional_string(params, "ensure_active_window_title") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let ensure_class = match parse_optional_string(params, "ensure_active_window_class") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };

    if let Err(err) = ensure_active_window(ensure_title.as_deref(), ensure_class.as_deref()) {
        return Response::error(err);
    }

    let (resolved_scale_x, resolved_scale_y) =
        resolve_click_scale(display_id, scale_x, scale_y);
    let (scaled_x, scaled_y) = apply_click_scale(x, y, resolved_scale_x, resolved_scale_y);

    if let Err(err) = perform_mouse_click(scaled_x, scaled_y, button) {
        return Response::error(err);
    }

    Response::success(json!({
        "clicked": true,
        "x": scaled_x,
        "y": scaled_y,
        "button": button_name(button),
        "scale_x": resolved_scale_x,
        "scale_y": resolved_scale_y,
    }))
}

#[derive(Clone, Copy, Debug)]
enum ClickButton {
    Left,
    Right,
    Middle,
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

fn parse_required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn parse_required_input_text(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn parse_optional_bool(params: &Value, field: &str) -> Result<bool, String> {
    let Some(raw) = params.get(field) else {
        return Ok(false);
    };
    raw.as_bool()
        .ok_or_else(|| format!("{field} must be a boolean"))
}

fn parse_optional_string(params: &Value, field: &str) -> Result<Option<String>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_str()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} must be a non-empty string"))?;
    Ok(Some(value))
}

fn parse_optional_usize(params: &Value, field: &str) -> Result<Option<usize>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    let value = raw
        .as_u64()
        .ok_or_else(|| format!("{field} must be a non-negative integer"))?;
    usize::try_from(value)
        .map(Some)
        .map_err(|_| format!("{field} is too large"))
}

fn parse_optional_f64(params: &Value, field: &str) -> Result<Option<f64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_f64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a number"))
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

fn ensure_active_window(expected_title: Option<&str>, expected_class: Option<&str>) -> Result<(), String> {
    if expected_title.is_none() && expected_class.is_none() {
        return Ok(());
    }

    #[cfg(target_os = "linux")]
    {
        if is_wayland_session() {
            return Err("active window verification is not supported on Wayland".to_string());
        }
        let active_id = command_stdout(Command::new("xdotool").arg("getactivewindow"))?;
        let active_id = parse_window_id(active_id.trim())
            .ok_or_else(|| "parse active window id failed".to_string())?;
        let list = command_stdout(Command::new("wmctrl").args(["-lxp"]))?;
        let (title, class_name) = find_window_by_id(&list, active_id)
            .ok_or_else(|| "active window is not found in wmctrl list".to_string())?;

        if let Some(expected) = expected_title {
            if !contains_case_insensitive(&title, expected) {
                return Err(format!(
                    "active window title mismatch: expected to contain {expected:?}, got {title:?}"
                ));
            }
        }
        if let Some(expected) = expected_class {
            if !contains_case_insensitive(&class_name, expected) {
                return Err(format!(
                    "active window class mismatch: expected to contain {expected:?}, got {class_name:?}"
                ));
            }
        }
        return Ok(());
    }

    #[cfg(not(target_os = "linux"))]
    {
        let _ = expected_title;
        let _ = expected_class;
        Err("active window verification is not supported on this platform".to_string())
    }
}

#[cfg(target_os = "linux")]
fn is_wayland_session() -> bool {
    matches!(
        env::var("XDG_SESSION_TYPE").ok().as_deref(),
        Some("wayland") | Some("Wayland")
    ) || env::var("WAYLAND_DISPLAY").map(|value| !value.is_empty()).unwrap_or(false)
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

#[cfg(target_os = "linux")]
fn find_window_by_id(listing: &str, window_id: u64) -> Option<(String, String)> {
    for line in listing.lines() {
        let mut parts = line.split_whitespace();
        let window_id_raw = parts.next()?;
        let _desktop = parts.next();
        let _pid = parts.next();
        let _host = parts.next();
        let class_name = parts.next()?;
        let title = parts.collect::<Vec<_>>().join(" ");
        let parsed_id = parse_window_id(window_id_raw)?;
        if parsed_id == window_id {
            return Some((title, class_name.to_string()));
        }
    }
    None
}

#[cfg(target_os = "linux")]
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

#[cfg(target_os = "linux")]
fn contains_case_insensitive(haystack: &str, needle: &str) -> bool {
    haystack.to_ascii_lowercase().contains(&needle.to_ascii_lowercase())
}

fn button_name(button: ClickButton) -> &'static str {
    match button {
        ClickButton::Left => "left",
        ClickButton::Right => "right",
        ClickButton::Middle => "middle",
    }
}

fn normalize_input_text(text: &str) -> String {
    text.replace("\r\n", "\n").replace('\r', "\n")
}

fn encode_png(image: xcap::image::RgbaImage) -> Result<Vec<u8>, String> {
    let mut cursor = std::io::Cursor::new(Vec::new());
    xcap::image::DynamicImage::ImageRgba8(image)
        .write_to(&mut cursor, xcap::image::ImageFormat::Png)
        .map_err(|err| format!("encode png failed: {err}"))?;
    Ok(cursor.into_inner())
}

fn monitor_scale(monitor: &Monitor, width: u32, height: u32) -> (f64, f64) {
    let logical_width = monitor.width();
    let logical_height = monitor.height();
    let scale_x = if logical_width > 0 {
        f64::from(width) / f64::from(logical_width)
    } else {
        1.0
    };
    let scale_y = if logical_height > 0 {
        f64::from(height) / f64::from(logical_height)
    } else {
        1.0
    };
    (sanitize_scale(scale_x), sanitize_scale(scale_y))
}

fn sanitize_scale(value: f64) -> f64 {
    if value.is_finite() && value > 0.0 {
        value
    } else {
        1.0
    }
}

fn resolve_click_scale(
    display_id: Option<u32>,
    scale_x: Option<f64>,
    scale_y: Option<f64>,
) -> (f64, f64) {
    let mut resolved_x = scale_x.unwrap_or(0.0);
    let mut resolved_y = scale_y.unwrap_or(0.0);
    if (resolved_x <= 0.0 || !resolved_x.is_finite())
        || (resolved_y <= 0.0 || !resolved_y.is_finite())
    {
        if let Some(id) = display_id {
            if let Some((cached_x, cached_y)) = lookup_display_scale(id) {
                if resolved_x <= 0.0 || !resolved_x.is_finite() {
                    resolved_x = cached_x;
                }
                if resolved_y <= 0.0 || !resolved_y.is_finite() {
                    resolved_y = cached_y;
                }
            }
        }
    }
    (sanitize_scale(resolved_x), sanitize_scale(resolved_y))
}

fn apply_click_scale(x: i32, y: i32, scale_x: f64, scale_y: f64) -> (i32, i32) {
    let scaled_x = if scale_x.is_finite() && scale_x > 0.0 && (scale_x - 1.0).abs() > 1e-6 {
        ((x as f64) / scale_x).round() as i32
    } else {
        x
    };
    let scaled_y = if scale_y.is_finite() && scale_y > 0.0 && (scale_y - 1.0).abs() > 1e-6 {
        ((y as f64) / scale_y).round() as i32
    } else {
        y
    };
    (scaled_x, scaled_y)
}

#[cfg(target_os = "linux")]
fn perform_mouse_click(x: i32, y: i32, button: ClickButton) -> Result<(), String> {
    if is_wayland_session() {
        return perform_mouse_click_wayland(x, y, button);
    }
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

#[cfg(target_os = "linux")]
fn perform_mouse_click_wayland(x: i32, y: i32, button: ClickButton) -> Result<(), String> {
    let button_id = match button {
        ClickButton::Left => "1",
        ClickButton::Middle => "2",
        ClickButton::Right => "3",
    };

    let mut move_status = Command::new("ydotool")
        .args(["mousemove", "--absolute", &x.to_string(), &y.to_string()])
        .status();
    if let Ok(status) = move_status {
        if !status.success() {
            move_status = Command::new("ydotool")
                .args(["mousemove", &x.to_string(), &y.to_string()])
                .status();
        }
    }

    let status = move_status.map_err(|err| {
        if err.kind() == std::io::ErrorKind::NotFound {
            "ydotool is not installed (required on Wayland for mouse input)".to_string()
        } else {
            format!("spawn ydotool failed: {err}")
        }
    })?;
    if !status.success() {
        return Err(format!(
            "ydotool mousemove exited with status {status}; ensure ydotoold is running"
        ));
    }

    let status = Command::new("ydotool")
        .args(["click", button_id])
        .status()
        .map_err(|err| {
            if err.kind() == std::io::ErrorKind::NotFound {
                "ydotool is not installed (required on Wayland for mouse input)".to_string()
            } else {
                format!("spawn ydotool failed: {err}")
            }
        })?;
    if !status.success() {
        return Err(format!(
            "ydotool click exited with status {status}; ensure ydotoold is running"
        ));
    }
    Ok(())
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_text_input(text: &str, submit: bool) -> Result<(), String> {
    let mut enigo = Enigo::new();
    let lines: Vec<&str> = text.split('\n').collect();
    for (index, segment) in lines.iter().enumerate() {
        if !segment.is_empty() {
            enigo.key_sequence(segment);
        }
        if index + 1 < lines.len() || submit {
            enigo.key_click(Key::Return);
        }
    }
    Ok(())
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_text_input(_text: &str, _submit: bool) -> Result<(), String> {
    Err("text input is not supported on this platform".to_string())
}

#[cfg(target_os = "linux")]
fn perform_text_input(text: &str, submit: bool) -> Result<(), String> {
    if is_wayland_session() {
        return perform_text_input_wayland(text, submit);
    }
    let lines: Vec<&str> = text.split('\n').collect();
    for (index, segment) in lines.iter().enumerate() {
        if !segment.is_empty() {
            let status = Command::new("xdotool")
                .args(["type", "--clearmodifiers", "--delay", "0", "--", segment])
                .status()
                .map_err(|err| format!("spawn xdotool failed: {err}"))?;
            if !status.success() {
                return Err(format!(
                    "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
                ));
            }
        }
        if index + 1 < lines.len() || submit {
            let status = Command::new("xdotool")
                .args(["key", "--clearmodifiers", "Return"])
                .status()
                .map_err(|err| format!("spawn xdotool failed: {err}"))?;
            if !status.success() {
                return Err(format!(
                    "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
                ));
            }
        }
    }
    Ok(())
}

#[cfg(target_os = "linux")]
fn perform_text_input_wayland(text: &str, submit: bool) -> Result<(), String> {
    let lines: Vec<&str> = text.split('\n').collect();
    for (index, segment) in lines.iter().enumerate() {
        if !segment.is_empty() {
            let status = Command::new("wtype")
                .args(["--", segment])
                .status()
                .map_err(|err| {
                    if err.kind() == std::io::ErrorKind::NotFound {
                        "wtype is not installed (required on Wayland for text input)".to_string()
                    } else {
                        format!("spawn wtype failed: {err}")
                    }
                })?;
            if !status.success() {
                return Err(format!(
                    "wtype exited with status {status}; ensure Wayland session is active"
                ));
            }
        }
        if index + 1 < lines.len() || submit {
            let status = Command::new("wtype")
                .args(["-k", "Return"])
                .status()
                .map_err(|err| {
                    if err.kind() == std::io::ErrorKind::NotFound {
                        "wtype is not installed (required on Wayland for text input)".to_string()
                    } else {
                        format!("spawn wtype failed: {err}")
                    }
                })?;
            if !status.success() {
                return Err(format!(
                    "wtype exited with status {status}; ensure Wayland session is active"
                ));
            }
        }
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

#[cfg(test)]
mod tests {
    use super::{
        button_name, dispatch_action, handle_bash_exec, normalize_input_text, parse_click_button,
        parse_optional_bool, parse_required_input_text,
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
    fn handle_bash_exec_returns_error_when_command_is_missing() {
        let response = handle_bash_exec(&json!({}));
        assert_eq!(response.status, "error");
        assert_eq!(response.error, "command is required");
    }

    #[test]
    fn handle_bash_exec_returns_output() {
        let response = handle_bash_exec(&json!({"command": shell_echo_command("hi")}));
        assert_eq!(response.status, "success");
        assert_eq!(response.error, "");
        let stdout = response.payload["stdout"]
            .as_str()
            .expect("stdout should be a string");
        assert!(stdout.contains("hi"), "unexpected stdout: {stdout}");
        assert!(
            response.payload["stderr"]
                .as_str()
                .expect("stderr should be a string")
                .is_empty()
        );
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_times_out() {
        let response = handle_bash_exec(&json!({
            "command": "sleep 1",
            "timeout_ms": 10
        }));
        assert_eq!(response.status, "error");
        assert!(
            response.error.contains("timed out"),
            "unexpected error: {}",
            response.error
        );
    }

    #[test]
    fn dispatch_action_returns_none_for_unknown_host_action() {
        assert!(dispatch_action("LIST_FILES", &json!({})).is_none());
    }

    #[test]
    fn parse_required_input_text_preserves_spaces() {
        let text =
            parse_required_input_text(&json!({"text":"  hello  "}), "text").expect("must parse");
        assert_eq!(text, "  hello  ");
    }

    #[test]
    fn parse_required_input_text_rejects_empty_string() {
        let err = parse_required_input_text(&json!({"text":""}), "text").expect_err("must fail");
        assert_eq!(err, "text is required");
    }

    #[test]
    fn parse_optional_bool_defaults_false() {
        let value = parse_optional_bool(&json!({}), "submit").expect("bool should parse");
        assert!(!value);
    }

    #[test]
    fn normalize_input_text_flattens_crlf() {
        assert_eq!(normalize_input_text("a\r\nb\rc"), "a\nb\nc");
    }

    fn shell_echo_command(text: &str) -> String {
        if cfg!(target_os = "windows") {
            format!("echo {text}")
        } else {
            format!("printf '{text}'")
        }
    }
}
