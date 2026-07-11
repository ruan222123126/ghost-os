use super::window_guard::ensure_active_window;
use crate::Response;
use crate::display_scale::lookup_display_scale;
use crate::json_params::{optional_display_id, optional_i32, optional_string, required_i32};
use crate::screen::types::{sanitize_scale, unscale_coordinate};
use serde_json::{Value, json};
use std::thread;
use std::time::Duration;

#[cfg(target_os = "linux")]
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseButton, MouseControllable};

pub(crate) fn handle_mouse_drag(params: &Value) -> Response {
    let request = match parse_drag_request(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    if let Err(err) = ensure_active_window(
        request.ensure_title.as_deref(),
        request.ensure_class.as_deref(),
    ) {
        return Response::error(err);
    }

    let (start_x, start_y, scale_x, scale_y) =
        resolve_drag_point(request.start_x, request.start_y, request.display_id);
    let (end_x, end_y, _, _) = resolve_drag_point(request.end_x, request.end_y, request.display_id);
    if let Err(err) = perform_mouse_drag(start_x, start_y, end_x, end_y, request.duration_ms) {
        return Response::error(err);
    }

    Response::success(json!({
        "dragged": true,
        "start_x": start_x,
        "start_y": start_y,
        "end_x": end_x,
        "end_y": end_y,
        "duration_ms": request.duration_ms,
        "scale_x": scale_x,
        "scale_y": scale_y,
    }))
}

struct DragRequest {
    start_x: i32,
    start_y: i32,
    end_x: i32,
    end_y: i32,
    duration_ms: i32,
    display_id: Option<u32>,
    ensure_title: Option<String>,
    ensure_class: Option<String>,
}

fn parse_drag_request(params: &Value) -> Result<DragRequest, String> {
    let duration_ms = optional_i32(params, "duration_ms")?.unwrap_or(150);
    if duration_ms < 0 {
        return Err("duration_ms must be a non-negative integer".to_string());
    }
    Ok(DragRequest {
        start_x: required_i32(params, "start_x")?,
        start_y: required_i32(params, "start_y")?,
        end_x: required_i32(params, "end_x")?,
        end_y: required_i32(params, "end_y")?,
        duration_ms,
        display_id: optional_display_id(params)?,
        ensure_title: optional_string(params, "ensure_active_window_title")?,
        ensure_class: optional_string(params, "ensure_active_window_class")?,
    })
}

fn resolve_drag_point(x: i32, y: i32, display_id: Option<u32>) -> (i32, i32, f64, f64) {
    let (scale_x, scale_y) = display_id
        .and_then(lookup_display_scale)
        .map(|(sx, sy)| (sanitize_scale(sx), sanitize_scale(sy)))
        .unwrap_or((1.0, 1.0));
    (
        unscale_coordinate(x, scale_x),
        unscale_coordinate(y, scale_y),
        scale_x,
        scale_y,
    )
}

fn drag_duration(duration_ms: i32) -> Duration {
    Duration::from_millis(duration_ms.max(0) as u64)
}

fn map_xdotool_spawn_error(err: std::io::Error) -> String {
    if err.kind() == std::io::ErrorKind::NotFound {
        return "xdotool is not installed".to_string();
    }
    format!("spawn xdotool failed: {err}")
}

#[cfg(target_os = "linux")]
fn perform_mouse_drag(
    start_x: i32,
    start_y: i32,
    end_x: i32,
    end_y: i32,
    duration_ms: i32,
) -> Result<(), String> {
    if super::window_guard::is_wayland_session() {
        return Err("mouse drag is not supported on Wayland".to_string());
    }
    run_xdotool_drag_step(&[
        "mousemove",
        "--sync",
        &start_x.to_string(),
        &start_y.to_string(),
    ])?;
    run_xdotool_drag_step(&["mousedown", "1"])?;
    thread::sleep(drag_duration(duration_ms));
    run_xdotool_drag_step(&[
        "mousemove",
        "--sync",
        &end_x.to_string(),
        &end_y.to_string(),
    ])?;
    run_xdotool_drag_step(&["mouseup", "1"])?;
    Ok(())
}

#[cfg(target_os = "linux")]
fn run_xdotool_drag_step(args: &[&str]) -> Result<(), String> {
    let status = Command::new("xdotool")
        .args(args)
        .status()
        .map_err(map_xdotool_spawn_error)?;
    if status.success() {
        return Ok(());
    }
    Err(format!(
        "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
    ))
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_mouse_drag(
    start_x: i32,
    start_y: i32,
    end_x: i32,
    end_y: i32,
    duration_ms: i32,
) -> Result<(), String> {
    let mut enigo = Enigo::new();
    enigo.mouse_move_to(start_x, start_y);
    enigo.mouse_down(MouseButton::Left);
    thread::sleep(drag_duration(duration_ms));
    enigo.mouse_move_to(end_x, end_y);
    enigo.mouse_up(MouseButton::Left);
    Ok(())
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_mouse_drag(
    _start_x: i32,
    _start_y: i32,
    _end_x: i32,
    _end_y: i32,
    _duration_ms: i32,
) -> Result<(), String> {
    Err("mouse drag is not supported on this platform".to_string())
}

#[cfg(test)]
#[path = "mouse_drag_tests.rs"]
mod tests;
