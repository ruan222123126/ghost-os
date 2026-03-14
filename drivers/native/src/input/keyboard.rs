use super::window_guard::is_wayland_session;
use crate::Response;
use crate::json_params::{optional_bool, required_text};
use serde_json::{Value, json};
use std::process::Command;

#[cfg(any(target_os = "macos", target_os = "windows"))]
use enigo::{Enigo, Key, KeyboardControllable};

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

fn normalize_input_text(text: &str) -> String {
    text.replace("\r\n", "\n").replace('\r', "\n")
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

#[cfg(test)]
mod tests {
    use super::{handle_text_input, normalize_input_text};
    use serde_json::json;

    #[test]
    fn normalize_input_text_flattens_crlf() {
        assert_eq!(normalize_input_text("a\r\nb\rc"), "a\nb\nc");
    }

    #[test]
    fn handle_text_input_rejects_empty_string() {
        let response = handle_text_input(&json!({"text":""}));
        assert_eq!(response.status, "error");
        assert_eq!(response.error, "text is required");
    }
}
