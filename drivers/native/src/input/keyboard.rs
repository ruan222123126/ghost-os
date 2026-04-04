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
            type_text_with_xdotool(segment)?;
        }
        if index + 1 < lines.len() || submit {
            trigger_xdotool_key("Return")?;
        }
    }
    Ok(())
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
fn type_text_with_xdotool(text: &str) -> Result<(), String> {
    let mut command = Command::new("xdotool");
    command.args(["key", "--clearmodifiers", "--delay", "0"]);
    for keysym in text_to_xdotool_keysyms(text) {
        command.arg(keysym);
    }
    run_xdotool(command)
}

#[cfg(target_os = "linux")]
fn trigger_xdotool_key(keysym: &str) -> Result<(), String> {
    let mut command = Command::new("xdotool");
    command.args(["key", "--clearmodifiers", keysym]);
    run_xdotool(command)
}

#[cfg(target_os = "linux")]
fn run_xdotool(mut command: Command) -> Result<(), String> {
    let status = command
        .status()
        .map_err(|err| format!("spawn xdotool failed: {err}"))?;
    if status.success() {
        return Ok(());
    }
    Err(format!(
        "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
    ))
}

#[cfg(target_os = "linux")]
fn text_to_xdotool_keysyms(text: &str) -> Vec<String> {
    text.chars()
        .map(|ch| format!("U{:04X}", ch as u32))
        .collect()
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
mod tests {
    #[cfg(target_os = "linux")]
    use super::text_to_xdotool_keysyms;
    use super::{handle_text_input, normalize_hotkey_key, normalize_input_text, parse_hotkey_keys};
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

    #[test]
    fn normalize_hotkey_key_maps_common_modifiers() {
        assert_eq!(normalize_hotkey_key("CTRL").expect("must parse"), "ctrl");
        assert_eq!(normalize_hotkey_key("enter").expect("must parse"), "Return");
    }

    #[test]
    fn parse_hotkey_keys_requires_array() {
        let err = parse_hotkey_keys(&json!({"keys":["CTRL","L"]})).expect("must parse");
        assert_eq!(err, vec!["ctrl".to_string(), "L".to_string()]);
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn text_to_xdotool_keysyms_encodes_mixed_language() {
        let got = text_to_xdotool_keysyms("我是 Ghost-OS 的 AI 助手");
        assert_eq!(
            got,
            vec![
                "U6211", "U662F", "U0020", "U0047", "U0068", "U006F", "U0073", "U0074", "U002D",
                "U004F", "U0053", "U0020", "U7684", "U0020", "U0041", "U0049", "U0020", "U52A9",
                "U624B",
            ]
        );
    }
}
