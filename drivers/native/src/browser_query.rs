use reqwest::blocking::Client as HttpClient;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::process::Command;
use std::time::Duration;

use crate::Response;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BrowserQueryKind {
    ActiveTab,
    WindowList,
    Url,
}

#[derive(Debug)]
struct BrowserQueryOptions {
    query_type: BrowserQueryKind,
    debug_host: String,
    debug_port: Option<u16>,
    include_non_browser: bool,
}

#[derive(Clone, Debug, Serialize)]
struct BrowserWindowInfo {
    window_id: String,
    title: String,
    class_name: String,
    pid: Option<i32>,
    is_active: bool,
    is_browser: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct CdpTargetInfo {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    url: String,
    #[serde(default, rename = "type")]
    target_type: String,
    #[serde(skip)]
    debug_port: u16,
}

pub(crate) fn handle_browser_query(params: &Value) -> Response {
    let options = match parse_browser_query_options(params) {
        Ok(options) => options,
        Err(err) => return Response::error(err),
    };

    let mut windows_error: Option<String> = None;
    let windows = match query_windows(options.include_non_browser) {
        Ok(windows) => windows,
        Err(err) => {
            windows_error = Some(err);
            Vec::new()
        }
    };

    let mut cdp_error: Option<String> = None;
    let cdp_targets = match query_cdp_targets(&options.debug_host, options.debug_port) {
        Ok(targets) => targets,
        Err(err) => {
            cdp_error = Some(err);
            Vec::new()
        }
    };

    match options.query_type {
        BrowserQueryKind::WindowList => {
            if !windows.is_empty() {
                return Response::success(json!({
                    "status": "ok",
                    "query_type": "window_list",
                    "count": windows.len(),
                    "windows": windows,
                }));
            }
            if !cdp_targets.is_empty() {
                return Response::success(json!({
                    "status": "ok",
                    "query_type": "window_list",
                    "count": 0,
                    "windows": [],
                    "tab_count": cdp_targets.len(),
                    "tabs": cdp_targets,
                    "source": "cdp",
                }));
            }
            Response::error(compose_browser_query_error(
                "window_list query failed",
                windows_error,
                cdp_error,
            ))
        }
        BrowserQueryKind::ActiveTab => {
            let active_window = pick_active_window(&windows);
            let matched_tab = active_window
                .as_ref()
                .and_then(|window| match_target_for_window(window, &cdp_targets))
                .or_else(|| cdp_targets.first());

            if active_window.is_none() && matched_tab.is_none() {
                return Response::error(compose_browser_query_error(
                    "active_tab query failed",
                    windows_error,
                    cdp_error,
                ));
            }

            let source = match (active_window.is_some(), matched_tab.is_some()) {
                (true, true) => "window+cdp",
                (true, false) => "window",
                (false, true) => "cdp",
                (false, false) => "unknown",
            };

            Response::success(json!({
                "status": "ok",
                "query_type": "active_tab",
                "window": active_window,
                "tab": matched_tab,
                "source": source,
            }))
        }
        BrowserQueryKind::Url => {
            let tab = pick_active_window(&windows)
                .and_then(|window| match_target_for_window(&window, &cdp_targets))
                .or_else(|| cdp_targets.first());

            let Some(tab) = tab else {
                return Response::error(compose_browser_query_error(
                    "url query failed: no CDP tab is available",
                    windows_error,
                    cdp_error,
                ));
            };

            let url = tab.url.trim();
            if url.is_empty() {
                return Response::error("url query failed: active tab URL is empty".to_string());
            }

            Response::success(json!({
                "status": "ok",
                "query_type": "url",
                "url": url,
                "title": tab.title,
                "tab_id": tab.id,
                "debug_port": tab.debug_port,
                "source": "cdp",
            }))
        }
    }
}

fn parse_browser_query_options(params: &Value) -> Result<BrowserQueryOptions, String> {
    let query_type_raw = params
        .get("query_type")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("active_tab")
        .to_ascii_lowercase();

    let query_type = match query_type_raw.as_str() {
        "active_tab" => BrowserQueryKind::ActiveTab,
        "window_list" => BrowserQueryKind::WindowList,
        "url" => BrowserQueryKind::Url,
        _ => {
            return Err(format!(
                "query_type must be one of: active_tab, window_list, url; got {query_type_raw}"
            ));
        }
    };

    let debug_host = params
        .get("debug_host")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("127.0.0.1")
        .to_string();

    let debug_port = parse_optional_u16(params, "debug_port")?;
    let include_non_browser = params
        .get("include_non_browser")
        .and_then(Value::as_bool)
        .unwrap_or(false);

    Ok(BrowserQueryOptions {
        query_type,
        debug_host,
        debug_port,
        include_non_browser,
    })
}

fn parse_optional_u16(params: &Value, field: &str) -> Result<Option<u16>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    let number = raw
        .as_u64()
        .ok_or_else(|| format!("{field} must be a non-negative integer"))?;
    let value = u16::try_from(number).map_err(|_| format!("{field} must be <= 65535"))?;
    if value == 0 {
        return Err(format!("{field} must be >= 1"));
    }
    Ok(Some(value))
}

fn compose_browser_query_error(
    prefix: &str,
    windows_error: Option<String>,
    cdp_error: Option<String>,
) -> String {
    match (windows_error, cdp_error) {
        (Some(windows), Some(cdp)) => format!("{prefix}: {windows}; {cdp}"),
        (Some(windows), None) => format!("{prefix}: {windows}"),
        (None, Some(cdp)) => format!("{prefix}: {cdp}"),
        (None, None) => prefix.to_string(),
    }
}

fn pick_active_window(windows: &[BrowserWindowInfo]) -> Option<BrowserWindowInfo> {
    windows
        .iter()
        .find(|window| window.is_active)
        .cloned()
        .or_else(|| windows.first().cloned())
}

fn match_target_for_window<'a>(
    window: &BrowserWindowInfo,
    targets: &'a [CdpTargetInfo],
) -> Option<&'a CdpTargetInfo> {
    let normalized_window_title = normalize_window_title(&window.title);
    if !normalized_window_title.is_empty() {
        for target in targets {
            let normalized_target_title = normalize_window_title(&target.title);
            if normalized_target_title.is_empty() {
                continue;
            }
            if normalized_window_title == normalized_target_title
                || normalized_window_title.contains(&normalized_target_title)
                || normalized_target_title.contains(&normalized_window_title)
            {
                return Some(target);
            }
        }
    }

    targets.iter().find(|target| !target.url.trim().is_empty())
}

fn normalize_window_title(raw: &str) -> String {
    let mut value = raw.trim().to_ascii_lowercase();
    for suffix in [
        " - google chrome",
        " - chromium",
        " - mozilla firefox",
        " - brave",
        " - microsoft edge",
        " - opera",
        " - vivaldi",
        " - safari",
        " - arc",
    ] {
        if value.ends_with(suffix) {
            value.truncate(value.len().saturating_sub(suffix.len()));
            break;
        }
    }

    value
        .split_whitespace()
        .filter(|part| !part.is_empty())
        .collect::<Vec<_>>()
        .join(" ")
}

fn query_cdp_targets(
    debug_host: &str,
    debug_port: Option<u16>,
) -> Result<Vec<CdpTargetInfo>, String> {
    const DEFAULT_PORTS: [u16; 4] = [9222, 9223, 9229, 9333];

    let ports = if let Some(port) = debug_port {
        vec![port]
    } else {
        DEFAULT_PORTS.to_vec()
    };

    let client = HttpClient::builder()
        .timeout(Duration::from_millis(450))
        .build()
        .map_err(|err| format!("create HTTP client failed: {err}"))?;

    let mut errors = Vec::new();
    for port in ports {
        let endpoint = format!("http://{debug_host}:{port}/json/list");
        let response = match client.get(&endpoint).send() {
            Ok(response) => response,
            Err(err) => {
                errors.push(format!("{endpoint}: {err}"));
                continue;
            }
        };

        if !response.status().is_success() {
            errors.push(format!("{endpoint}: HTTP {}", response.status()));
            continue;
        }

        let body = match response.text() {
            Ok(body) => body,
            Err(err) => {
                errors.push(format!("{endpoint}: read body failed: {err}"));
                continue;
            }
        };
        let mut targets: Vec<CdpTargetInfo> = match serde_json::from_str(&body) {
            Ok(targets) => targets,
            Err(err) => {
                errors.push(format!("{endpoint}: decode json failed: {err}"));
                continue;
            }
        };

        targets.retain(|target| {
            target.target_type.eq_ignore_ascii_case("page")
                && !target.url.starts_with("devtools://")
                && !target.url.starts_with("chrome://")
        });

        if targets.is_empty() {
            errors.push(format!("{endpoint}: no page target"));
            continue;
        }

        for target in &mut targets {
            target.debug_port = port;
        }
        return Ok(targets);
    }

    if errors.is_empty() {
        return Err("CDP target discovery failed".to_string());
    }
    Err(format!(
        "CDP target discovery failed; ensure browser remote debugging is enabled (example: --remote-debugging-port=9222). details: {}",
        errors.join(" | ")
    ))
}

#[cfg(target_os = "linux")]
fn query_windows(include_non_browser: bool) -> Result<Vec<BrowserWindowInfo>, String> {
    let active_window_id = query_active_window_id_linux().ok();
    let output = Command::new("wmctrl")
        .args(["-lxp"])
        .output()
        .map_err(|err| format!("spawn wmctrl failed: {err}"))?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        let detail = stderr.lines().next().unwrap_or("").trim();
        if detail.is_empty() {
            return Err("wmctrl returned non-zero status".to_string());
        }
        return Err(format!("wmctrl failed: {detail}"));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut windows = Vec::new();
    for line in stdout.lines() {
        let mut parts = line.split_whitespace();
        let Some(window_id_raw) = parts.next() else {
            continue;
        };
        let _desktop = parts.next();
        let pid_raw = parts.next();
        let _host = parts.next();
        let Some(class_name_raw) = parts.next() else {
            continue;
        };
        let title = parts.collect::<Vec<_>>().join(" ");

        let class_name = class_name_raw.to_string();
        let is_browser = is_browser_window_class(&class_name);
        if !include_non_browser && !is_browser {
            continue;
        }

        let window_id_num = parse_window_id(window_id_raw).unwrap_or(0);
        let is_active = active_window_id
            .map(|active_id| active_id == window_id_num)
            .unwrap_or(false);
        windows.push(BrowserWindowInfo {
            window_id: window_id_raw.to_string(),
            title,
            class_name,
            pid: pid_raw.and_then(|value| value.parse::<i32>().ok()),
            is_active,
            is_browser,
        });
    }

    Ok(windows)
}

#[cfg(not(target_os = "linux"))]
fn query_windows(_include_non_browser: bool) -> Result<Vec<BrowserWindowInfo>, String> {
    Err("window_list query is currently supported on Linux only".to_string())
}

#[cfg(target_os = "linux")]
fn query_active_window_id_linux() -> Result<u64, String> {
    let output = command_stdout(Command::new("xdotool").arg("getactivewindow"))?;
    output
        .trim()
        .parse::<u64>()
        .map_err(|err| format!("parse active window id failed: {err}"))
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

fn is_browser_window_class(class_name: &str) -> bool {
    let normalized = class_name.to_ascii_lowercase();
    [
        "firefox", "chrome", "chromium", "brave", "edge", "opera", "vivaldi", "safari", "arc",
    ]
    .iter()
    .any(|keyword| normalized.contains(keyword))
}

#[cfg(test)]
mod tests {
    use super::{
        BrowserQueryKind, BrowserWindowInfo, CdpTargetInfo, is_browser_window_class,
        match_target_for_window, normalize_window_title, parse_browser_query_options,
        parse_window_id,
    };
    use serde_json::json;

    #[test]
    fn parse_browser_query_options_defaults_to_active_tab() {
        let options = parse_browser_query_options(&json!({})).expect("must parse");
        assert_eq!(options.query_type, BrowserQueryKind::ActiveTab);
        assert_eq!(options.debug_host, "127.0.0.1");
        assert_eq!(options.debug_port, None);
        assert!(!options.include_non_browser);
    }

    #[test]
    fn parse_browser_query_options_rejects_unknown_query_type() {
        let err =
            parse_browser_query_options(&json!({"query_type":"unknown"})).expect_err("must fail");
        assert!(err.contains("query_type must be one of"));
    }

    #[test]
    fn parse_window_id_supports_hex_and_decimal() {
        assert_eq!(parse_window_id("0x10"), Some(16));
        assert_eq!(parse_window_id("15"), Some(15));
        assert_eq!(parse_window_id(""), None);
    }

    #[test]
    fn normalize_window_title_strips_browser_suffix() {
        assert_eq!(
            normalize_window_title("Ghost-OS - Google Chrome"),
            "ghost-os"
        );
        assert_eq!(normalize_window_title("  Example   Tab   "), "example tab");
    }

    #[test]
    fn is_browser_window_class_matches_known_browsers() {
        assert!(is_browser_window_class("Navigator.Firefox"));
        assert!(is_browser_window_class("google-chrome.Google-chrome"));
        assert!(!is_browser_window_class("code.Code"));
    }

    #[test]
    fn match_target_for_window_uses_title_similarity() {
        let window = BrowserWindowInfo {
            window_id: "0x1".to_string(),
            title: "Ghost-OS Dashboard - Google Chrome".to_string(),
            class_name: "google-chrome.Google-chrome".to_string(),
            pid: Some(1),
            is_active: true,
            is_browser: true,
        };
        let targets = vec![
            CdpTargetInfo {
                id: "tab-a".to_string(),
                title: "Other".to_string(),
                url: "https://example.com".to_string(),
                target_type: "page".to_string(),
                debug_port: 9222,
            },
            CdpTargetInfo {
                id: "tab-b".to_string(),
                title: "Ghost-OS Dashboard".to_string(),
                url: "https://ghost.local/dashboard".to_string(),
                target_type: "page".to_string(),
                debug_port: 9222,
            },
        ];

        let matched = match_target_for_window(&window, &targets).expect("must match");
        assert_eq!(matched.id, "tab-b");
    }
}
