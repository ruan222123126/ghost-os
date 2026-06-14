#[cfg(target_os = "linux")]
use std::env;
use std::process::Command;

use crate::Response;
use serde_json::{Value, json};

pub(crate) fn ensure_active_window(
    expected_title: Option<&str>,
    expected_class: Option<&str>,
) -> Result<(), String> {
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
        let (title, class_name) = lookup_active_window(active_id)?;

        if let Some(expected) = expected_title
            && !contains_case_insensitive(&title, expected)
        {
            return Err(format!(
                "active window title mismatch: expected to contain {expected:?}, got {title:?}"
            ));
        }
        if let Some(expected) = expected_class
            && !contains_case_insensitive(&class_name, expected)
        {
            return Err(format!(
                "active window class mismatch: expected to contain {expected:?}, got {class_name:?}"
            ));
        }
        Ok(())
    }

    #[cfg(not(target_os = "linux"))]
    {
        let _ = expected_title;
        let _ = expected_class;
        Err("active window verification is not supported on this platform".to_string())
    }
}

pub(crate) fn handle_active_window_info(_params: &Value) -> Response {
    match active_window_info() {
        Ok((title, class_name)) => Response::success(json!({
            "title": title,
            "class": class_name,
        })),
        Err(err) => Response::error(err),
    }
}

pub(crate) fn active_window_info() -> Result<(String, String), String> {
    #[cfg(target_os = "linux")]
    {
        if is_wayland_session() {
            return Err("active window info is not supported on Wayland".to_string());
        }
        let active_id = command_stdout(Command::new("xdotool").arg("getactivewindow"))?;
        let active_id = parse_window_id(active_id.trim())
            .ok_or_else(|| "parse active window id failed".to_string())?;
        lookup_active_window(active_id)
    }

    #[cfg(not(target_os = "linux"))]
    {
        Err("active window info is not supported on this platform".to_string())
    }
}

#[cfg(target_os = "linux")]
pub(crate) fn is_wayland_session() -> bool {
    matches!(
        env::var("XDG_SESSION_TYPE").ok().as_deref(),
        Some("wayland") | Some("Wayland")
    ) || env::var("WAYLAND_DISPLAY")
        .map(|value| !value.is_empty())
        .unwrap_or(false)
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
fn lookup_active_window(active_id: u64) -> Result<(String, String), String> {
    let list = command_stdout(Command::new("wmctrl").args(["-lxp"]))?;
    find_window_by_id(&list, active_id)
        .ok_or_else(|| "active window is not found in wmctrl list".to_string())
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
    haystack
        .to_ascii_lowercase()
        .contains(&needle.to_ascii_lowercase())
}

#[cfg(test)]
mod tests {
    use super::parse_window_id;

    #[test]
    fn parse_window_id_supports_decimal_and_hex() {
        assert_eq!(parse_window_id("12"), Some(12));
        assert_eq!(parse_window_id("0x10"), Some(16));
    }
}
