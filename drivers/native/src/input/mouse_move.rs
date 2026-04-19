use super::window_guard::ensure_active_window;
use crate::Response;
use crate::display_scale::lookup_display_scale;
use crate::json_params::{optional_display_id, optional_string, required_i32};
use crate::screen::types::{sanitize_scale, unscale_coordinate};
use serde_json::{Value, json};

#[cfg(target_os = "linux")]
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseControllable};

pub(crate) fn handle_mouse_move(params: &Value) -> Response {
    let x = match required_i32(params, "x") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let y = match required_i32(params, "y") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let display_id = match optional_display_id(params) {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let ensure_title = match optional_string(params, "ensure_active_window_title") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let ensure_class = match optional_string(params, "ensure_active_window_class") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    if let Err(err) = ensure_active_window(ensure_title.as_deref(), ensure_class.as_deref()) {
        return Response::error(err);
    }

    let (scaled_x, scaled_y, scale_x, scale_y) = resolve_target_point(x, y, display_id);
    if let Err(err) = perform_mouse_move(scaled_x, scaled_y) {
        return Response::error(err);
    }
    Response::success(json!({
        "moved": true,
        "x": scaled_x,
        "y": scaled_y,
        "scale_x": scale_x,
        "scale_y": scale_y,
    }))
}

fn resolve_target_point(x: i32, y: i32, display_id: Option<u32>) -> (i32, i32, f64, f64) {
    let (scale_x, scale_y) = display_id
        .and_then(lookup_display_scale)
        .map(|(sx, sy)| (sanitize_scale(sx), sanitize_scale(sy)))
        .unwrap_or((1.0, 1.0));
    let scaled_x = unscale_coordinate(x, scale_x);
    let scaled_y = unscale_coordinate(y, scale_y);
    (scaled_x, scaled_y, scale_x, scale_y)
}

#[cfg(target_os = "linux")]
fn perform_mouse_move(x: i32, y: i32) -> Result<(), String> {
    if super::window_guard::is_wayland_session() {
        return perform_mouse_move_wayland(x, y);
    }
    let status = Command::new("xdotool")
        .args(["mousemove", "--sync", &x.to_string(), &y.to_string()])
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
fn perform_mouse_move_wayland(x: i32, y: i32) -> Result<(), String> {
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
    let status = move_status.map_err(|err| map_spawn_ydotool_error(err))?;
    if status.success() {
        return Ok(());
    }
    Err(format!(
        "ydotool mousemove exited with status {status}; ensure ydotoold is running"
    ))
}

#[cfg(target_os = "linux")]
fn map_spawn_ydotool_error(err: std::io::Error) -> String {
    if err.kind() == std::io::ErrorKind::NotFound {
        return "ydotool is not installed (required on Wayland for mouse input)".to_string();
    }
    format!("spawn ydotool failed: {err}")
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_mouse_move(x: i32, y: i32) -> Result<(), String> {
    let mut enigo = Enigo::new();
    enigo.mouse_move_to(x, y);
    Ok(())
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_mouse_move(_x: i32, _y: i32) -> Result<(), String> {
    Err("mouse move is not supported on this platform".to_string())
}

#[cfg(test)]
#[path = "mouse_move_tests.rs"]
mod tests;
