use crate::Response;
use crate::display_scale::record_display_scale;
use crate::screen::types::{sanitize_scale, scale_coordinate};
use serde::Serialize;
use serde_json::Value;
use xcap::Monitor;

#[cfg(target_os = "linux")]
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseControllable};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct RawMousePosition {
    x: i32,
    y: i32,
}

#[derive(Clone, Copy, Debug, PartialEq, Serialize)]
struct MousePositionPayload {
    x: i32,
    y: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    display_id: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    scale_x: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    scale_y: Option<f64>,
}

pub(crate) fn handle_mouse_position(_params: &Value) -> Response {
    let raw = match read_raw_mouse_position() {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let payload = resolve_mouse_position_payload(raw);
    match serde_json::to_value(payload) {
        Ok(value) => Response::success(value),
        Err(err) => Response::error(format!("encode mouse position payload failed: {err}")),
    }
}

fn resolve_mouse_position_payload(raw: RawMousePosition) -> MousePositionPayload {
    let Some(monitor) = monitor_from_point(raw) else {
        return MousePositionPayload {
            x: raw.x,
            y: raw.y,
            display_id: None,
            scale_x: None,
            scale_y: None,
        };
    };
    let scale_x = sanitize_scale(f64::from(monitor.scale_factor()));
    let scale_y = scale_x;
    record_display_scale(monitor.id(), scale_x, scale_y);
    MousePositionPayload {
        x: scale_coordinate(raw.x, scale_x),
        y: scale_coordinate(raw.y, scale_y),
        display_id: Some(monitor.id()),
        scale_x: Some(scale_x),
        scale_y: Some(scale_y),
    }
}

fn monitor_from_point(raw: RawMousePosition) -> Option<Monitor> {
    Monitor::from_point(raw.x, raw.y).ok()
}

#[cfg(target_os = "linux")]
fn read_raw_mouse_position() -> Result<RawMousePosition, String> {
    if super::window_guard::is_wayland_session() {
        return Err("mouse position query is not supported on Wayland".to_string());
    }
    let output = Command::new("xdotool")
        .args(["getmouselocation", "--shell"])
        .output()
        .map_err(map_spawn_xdotool_error)?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let detail = stderr.lines().next().unwrap_or("").trim();
        if detail.is_empty() {
            return Err(format!("xdotool exited with status {}", output.status));
        }
        return Err(detail.to_string());
    }
    parse_xdotool_mouse_position(&String::from_utf8_lossy(&output.stdout))
}

#[cfg(target_os = "linux")]
fn parse_xdotool_mouse_position(output: &str) -> Result<RawMousePosition, String> {
    let mut x = None;
    let mut y = None;
    for line in output.lines() {
        let Some((key, value)) = line.split_once('=') else {
            continue;
        };
        let parsed = value
            .trim()
            .parse::<i32>()
            .map_err(|err| format!("parse xdotool {key} failed: {err}"))?;
        match key.trim() {
            "X" => x = Some(parsed),
            "Y" => y = Some(parsed),
            _ => {}
        }
    }
    match (x, y) {
        (Some(x), Some(y)) => Ok(RawMousePosition { x, y }),
        _ => Err("xdotool getmouselocation missing X or Y".to_string()),
    }
}

#[cfg(target_os = "linux")]
fn map_spawn_xdotool_error(err: std::io::Error) -> String {
    if err.kind() == std::io::ErrorKind::NotFound {
        return "xdotool is not installed".to_string();
    }
    format!("spawn xdotool failed: {err}")
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn read_raw_mouse_position() -> Result<RawMousePosition, String> {
    let mut enigo = Enigo::new();
    let (x, y) = enigo.mouse_location();
    Ok(RawMousePosition { x, y })
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn read_raw_mouse_position() -> Result<RawMousePosition, String> {
    Err("mouse position query is not supported on this platform".to_string())
}

#[cfg(test)]
#[path = "mouse_position_tests.rs"]
mod tests;
