use super::window_guard::ensure_active_window;
use crate::Response;
use crate::display_scale::lookup_display_scale;
use crate::json_params::{optional_display_id, optional_string, required_i32};
use crate::screen::types::{sanitize_scale, unscale_coordinate};
use serde_json::{Value, json};
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, MouseButton, MouseControllable};

pub(crate) fn handle_mouse_click(params: &Value) -> Response {
    handle_mouse_click_request(params, None, 1)
}

pub(crate) fn handle_mouse_double_click(params: &Value) -> Response {
    handle_mouse_click_request(params, Some(ClickButton::Left), 2)
}

pub(crate) fn handle_mouse_right_click(params: &Value) -> Response {
    handle_mouse_click_request(params, Some(ClickButton::Right), 1)
}

fn handle_mouse_click_request(
    params: &Value,
    forced_button: Option<ClickButton>,
    repeat: usize,
) -> Response {
    let x = match required_i32(params, "x") {
        Ok(x) => x,
        Err(err) => return Response::error(err),
    };
    let y = match required_i32(params, "y") {
        Ok(y) => y,
        Err(err) => return Response::error(err),
    };
    let button = match forced_button {
        Some(button) => button,
        None => match parse_click_button(params) {
            Ok(button) => button,
            Err(err) => return Response::error(err),
        },
    };
    let display_id = match optional_display_id(params) {
        Ok(display_id) => display_id,
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

    let (scaled_x, scaled_y, resolved_scale_x, resolved_scale_y) =
        resolve_target_point(x, y, display_id);
    if let Err(err) = perform_mouse_click(scaled_x, scaled_y, button, repeat) {
        return Response::error(err);
    }

    Response::success(json!({
        "clicked": true,
        "x": scaled_x,
        "y": scaled_y,
        "button": button_name(button),
        "repeat": repeat,
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

fn resolve_target_point(x: i32, y: i32, display_id: Option<u32>) -> (i32, i32, f64, f64) {
    let (scale_x, scale_y) = display_id
        .and_then(lookup_display_scale)
        .map(|(sx, sy)| (sanitize_scale(sx), sanitize_scale(sy)))
        .unwrap_or((1.0, 1.0));
    let scaled_x = unscale_coordinate(x, scale_x);
    let scaled_y = unscale_coordinate(y, scale_y);
    (scaled_x, scaled_y, scale_x, scale_y)
}

fn button_name(button: ClickButton) -> &'static str {
    match button {
        ClickButton::Left => "left",
        ClickButton::Right => "right",
        ClickButton::Middle => "middle",
    }
}

fn normalized_repeat(repeat: usize) -> usize {
    repeat.max(1)
}

fn map_spawn_error(program: &str, err: std::io::Error, not_found_message: Option<&str>) -> String {
    if err.kind() == std::io::ErrorKind::NotFound {
        return not_found_message
            .map(str::to_string)
            .unwrap_or_else(|| format!("{program} is not installed"));
    }
    format!("spawn {program} failed: {err}")
}

#[cfg(target_os = "linux")]
fn perform_mouse_click(x: i32, y: i32, button: ClickButton, repeat: usize) -> Result<(), String> {
    if super::window_guard::is_wayland_session() {
        return perform_mouse_click_wayland(x, y, button, repeat);
    }
    let button_id = match button {
        ClickButton::Left => "1",
        ClickButton::Middle => "2",
        ClickButton::Right => "3",
    };
    let repeat_value = normalized_repeat(repeat).to_string();
    let status = Command::new("xdotool")
        .args([
            "mousemove",
            "--sync",
            &x.to_string(),
            &y.to_string(),
            "click",
            "--repeat",
            &repeat_value,
            button_id,
        ])
        .status()
        .map_err(|err| map_spawn_error("xdotool", err, None))?;
    if !status.success() {
        return Err(format!(
            "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
        ));
    }
    Ok(())
}

#[cfg(target_os = "linux")]
fn perform_mouse_click_wayland(
    x: i32,
    y: i32,
    button: ClickButton,
    repeat: usize,
) -> Result<(), String> {
    let button_id = match button {
        ClickButton::Left => "1",
        ClickButton::Middle => "2",
        ClickButton::Right => "3",
    };
    ydotool_move(x, y)?;
    ydotool_click(button_id, normalized_repeat(repeat))?;
    Ok(())
}

#[cfg(target_os = "linux")]
fn ydotool_move(x: i32, y: i32) -> Result<(), String> {
    let mut move_status = Command::new("ydotool")
        .args(["mousemove", "--absolute", &x.to_string(), &y.to_string()])
        .status();
    if let Ok(status) = move_status
        && !status.success()
    {
        move_status = Command::new("ydotool")
            .args(["mousemove", &x.to_string(), &y.to_string()])
            .status();
    }
    let status = move_status.map_err(map_spawn_ydotool_error)?;
    if status.success() {
        return Ok(());
    }
    Err(format!(
        "ydotool mousemove exited with status {status}; ensure ydotoold is running"
    ))
}

#[cfg(target_os = "linux")]
fn ydotool_click(button_id: &str, repeat: usize) -> Result<(), String> {
    for _ in 0..repeat {
        let status = Command::new("ydotool")
            .args(["click", button_id])
            .status()
            .map_err(map_spawn_ydotool_error)?;
        if !status.success() {
            return Err(format!(
                "ydotool click exited with status {status}; ensure ydotoold is running"
            ));
        }
    }
    Ok(())
}

#[cfg(target_os = "linux")]
fn map_spawn_ydotool_error(err: std::io::Error) -> String {
    map_spawn_error(
        "ydotool",
        err,
        Some("ydotool is not installed (required on Wayland for mouse input)"),
    )
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_mouse_click(x: i32, y: i32, button: ClickButton, repeat: usize) -> Result<(), String> {
    let mut enigo = Enigo::new();
    enigo.mouse_move_to(x, y);
    let native_button = match button {
        ClickButton::Left => MouseButton::Left,
        ClickButton::Right => MouseButton::Right,
        ClickButton::Middle => MouseButton::Middle,
    };
    for _ in 0..repeat.max(1) {
        enigo.mouse_click(native_button);
    }
    Ok(())
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_mouse_click(
    _x: i32,
    _y: i32,
    _button: ClickButton,
    _repeat: usize,
) -> Result<(), String> {
    Err("mouse click is not supported on this platform".to_string())
}

#[cfg(test)]
#[path = "mouse_tests.rs"]
mod tests;
