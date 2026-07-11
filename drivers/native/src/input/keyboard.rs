use super::ime_guard::X11InputMethodGuard;
use super::window_guard::is_wayland_session;
use crate::Response;
use crate::json_params::{optional_bool, required_text};
use serde_json::{Value, json};
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, Key, KeyboardControllable};
#[cfg(target_os = "linux")]
#[path = "keyboard_x11.rs"]
mod keyboard_x11;
#[cfg(target_os = "linux")]
use self::keyboard_x11::{active_window_id_for_xdotool, type_lines_with_xdotool};

pub(crate) fn handle_text_input(params: &Value) -> Response {
    let text = match required_text(params, "text") {
        Ok(text) => text,
        Err(err) => return Response::error(err),
    };
    let submit = match optional_bool(params, "submit") {
        Ok(value) => value.unwrap_or(false),
        Err(err) => return Response::error(err),
    };
    let normalized = normalize_input_text(&text);
    let ime_restore_error = match perform_text_input(&normalized, submit) {
        Ok(error) => error,
        Err(err) => return Response::error(err),
    };
    let mut payload = json!({
        "typed": true,
        "submitted": submit,
        "characters": normalized.chars().count(),
        "lines": normalized.split('\n').count(),
    });
    if let Some(error) = ime_restore_error {
        payload["ime_restore_error"] = json!(error);
    }
    Response::success(payload)
}

pub(crate) fn handle_key_hotkey(params: &Value) -> Response {
    let keys = match parse_hotkey_keys(params) {
        Ok(keys) => keys,
        Err(err) => return Response::error(err),
    };
    if let Err(err) = perform_hotkey(&keys) {
        return Response::error(err);
    }
    Response::success(json!({
        "pressed": true,
        "keys": keys,
    }))
}

fn normalize_input_text(text: &str) -> String {
    text.replace("\r\n", "\n").replace('\r', "\n")
}

fn parse_hotkey_keys(params: &Value) -> Result<Vec<String>, String> {
    let raw = params
        .get("keys")
        .and_then(Value::as_array)
        .ok_or_else(|| "keys is required".to_string())?;
    if raw.is_empty() {
        return Err("keys is required".to_string());
    }
    let mut out = Vec::with_capacity(raw.len());
    for value in raw {
        let key = value
            .as_str()
            .map(str::trim)
            .filter(|item| !item.is_empty())
            .ok_or_else(|| "keys must be a non-empty string array".to_string())?;
        out.push(normalize_hotkey_key(key)?);
    }
    Ok(out)
}

fn normalize_hotkey_key(key: &str) -> Result<String, String> {
    let upper = key.to_ascii_uppercase();
    let normalized = match upper.as_str() {
        "CTRL" | "CONTROL" => "ctrl",
        "SHIFT" => "shift",
        "ALT" | "OPTION" => "alt",
        "CMD" | "COMMAND" | "META" | "SUPER" | "WIN" => "super",
        "ENTER" | "RETURN" => "Return",
        "ESC" | "ESCAPE" => "Escape",
        "TAB" => "Tab",
        "SPACE" => "space",
        "BACKSPACE" => "BackSpace",
        "DELETE" => "Delete",
        "UP" | "ARROWUP" => "Up",
        "DOWN" | "ARROWDOWN" => "Down",
        "LEFT" | "ARROWLEFT" => "Left",
        "RIGHT" | "ARROWRIGHT" => "Right",
        key if key.len() == 1 => key,
        _ => return Err(format!("unsupported hotkey key {key:?}")),
    };
    Ok(normalized.to_string())
}

#[cfg(any(target_os = "macos", target_os = "windows"))]
fn perform_text_input(text: &str, submit: bool) -> Result<Option<String>, String> {
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
    Ok(None)
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn perform_text_input(_text: &str, _submit: bool) -> Result<Option<String>, String> {
    Err("text input is not supported on this platform".to_string())
}

#[cfg(target_os = "linux")]
fn perform_text_input(text: &str, submit: bool) -> Result<Option<String>, String> {
    if is_wayland_session() {
        perform_text_input_wayland(text, submit)?;
        return Ok(None);
    }
    let window_id = active_window_id_for_xdotool()?;
    let ime_guard = X11InputMethodGuard::activate()?;
    let typing_result = type_lines_with_xdotool(&window_id, text, submit);
    finalize_restore_result(typing_result, ime_guard.restore())
}

#[cfg(target_os = "linux")]
fn perform_hotkey(keys: &[String]) -> Result<(), String> {
    if is_wayland_session() {
        return Err("hotkey input is not supported on Wayland".to_string());
    }
    let combo = keys.join("+");
    let status = Command::new("xdotool")
        .args(["key", "--clearmodifiers", &combo])
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
fn finalize_restore_result(
    typing_result: Result<(), String>,
    restore_result: Result<(), String>,
) -> Result<Option<String>, String> {
    match (typing_result, restore_result) {
        (Err(type_err), Err(restore_err)) => Err(format!(
            "{type_err}; restore input method failed: {restore_err}"
        )),
        (Err(type_err), Ok(())) => Err(type_err),
        (Ok(()), Err(restore_err)) => {
            Ok(Some(format!("restore input method failed: {restore_err}")))
        }
        (Ok(()), Ok(())) => Ok(None),
    }
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

#[cfg(not(target_os = "linux"))]
fn perform_hotkey(_keys: &[String]) -> Result<(), String> {
    Err("hotkey input is not supported on this platform".to_string())
}
#[cfg(test)]
#[path = "keyboard_tests.rs"]
mod tests;
