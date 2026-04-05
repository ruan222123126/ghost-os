use crate::json_params::{optional_string, optional_usize};
use crate::sandbox::shell_tools::{
    bash_exec_impl, format_bash_exec_failure, resolve_shell_timeout,
};
use crate::sandbox::SandboxConfig;
use crate::Response;
use serde_json::{json, Value};

const DEFAULT_BROWSER_DEBUG_PORT: usize = 9222;
const MIN_BROWSER_DEBUG_PORT: usize = 1;
const MAX_BROWSER_DEBUG_PORT: usize = 65535;
const BROWSER_PROFILE_ROOT_DIR: &str = "/tmp/ghost-browser-control-profile";
const BROWSER_LOG_ROOT_DIR: &str = "/tmp/ghost-browser-control-log";
const DEFAULT_BROWSER_PROFILE_KEY: &str = "default";
const BROWSER_DISPLAY_MODE_BACKGROUND: &str = "background";
const BROWSER_DISPLAY_MODE_FOREGROUND: &str = "foreground";
const BROWSER_REUSE_MAX_TIME_SECONDS: usize = 1;

// Execution-layer OS details for auto-discovery stay here, not in Central.
const BROWSER_CANDIDATES: [&str; 5] = [
    "google-chrome",
    "google-chrome-stable",
    "chromium",
    "chromium-browser",
    "chrome",
];

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "BROWSER_LAUNCH" => Some(handle_browser_launch(params)),
        _ => None,
    }
}

fn handle_browser_launch(params: &Value) -> Response {
    let config = SandboxConfig::default();
    let timeout_ms = match optional_usize(params, "timeout_ms") {
        Ok(timeout_ms) => resolve_shell_timeout(&config, timeout_ms.map(|value| value as u64)),
        Err(err) => return Response::error(err),
    };
    let command = match browser_launch_command(params) {
        Ok(command) => command,
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

fn browser_launch_command(params: &Value) -> Result<String, String> {
    let display_mode = parse_display_mode(params)?;
    let debug_port = optional_debug_port(params)?;
    if let Some(command) = optional_string(params, "command")? {
        return Ok(command);
    }
    let launch_port = debug_port.unwrap_or(DEFAULT_BROWSER_DEBUG_PORT);
    let profile_key = browser_profile_key(params, launch_port)?;
    let profile_dir = browser_profile_dir(&profile_key);
    let log_path = browser_log_path(&profile_key);
    Ok(build_auto_browser_launch_command(
        launch_port,
        display_mode,
        &profile_dir,
        &log_path,
    ))
}

fn optional_debug_port(params: &Value) -> Result<Option<usize>, String> {
    let Some(port) = optional_usize(params, "debug_port")? else {
        return Ok(None);
    };
    if !(MIN_BROWSER_DEBUG_PORT..=MAX_BROWSER_DEBUG_PORT).contains(&port) {
        return Err(format!(
            "debug_port must be between {} and {}",
            MIN_BROWSER_DEBUG_PORT, MAX_BROWSER_DEBUG_PORT
        ));
    }
    Ok(Some(port))
}

fn parse_display_mode(params: &Value) -> Result<BrowserDisplayMode, String> {
    let Some(raw_mode) = optional_string(params, "display_mode")? else {
        return Ok(BrowserDisplayMode::Background);
    };
    BrowserDisplayMode::from_raw(&raw_mode)
}

fn browser_profile_key(params: &Value, debug_port: usize) -> Result<String, String> {
    if let Some(session_id) = optional_string(params, "session_id")? {
        return Ok(format!("session-{}", sanitize_path_component(&session_id)));
    }
    Ok(format!("port-{debug_port}"))
}

fn browser_profile_dir(profile_key: &str) -> String {
    format!(
        "{BROWSER_PROFILE_ROOT_DIR}/{}",
        sanitize_path_component(profile_key)
    )
}

fn browser_log_path(profile_key: &str) -> String {
    format!(
        "{BROWSER_LOG_ROOT_DIR}/{}.log",
        sanitize_path_component(profile_key)
    )
}

fn sanitize_path_component(raw: &str) -> String {
    let sanitized: String = raw
        .trim()
        .chars()
        .map(|ch| match ch {
            'a'..='z' | 'A'..='Z' | '0'..='9' | '-' | '_' | '.' => ch,
            _ => '_',
        })
        .collect();
    let trimmed = sanitized.trim_matches('_');
    if trimmed.is_empty() {
        return DEFAULT_BROWSER_PROFILE_KEY.to_string();
    }
    trimmed.to_string()
}

fn build_auto_browser_launch_command(
    debug_port: usize,
    display_mode: BrowserDisplayMode,
    profile_dir: &str,
    log_path: &str,
) -> String {
    format!(
        r#"set -euo pipefail
if command -v curl >/dev/null 2>&1; then
	if curl --silent --show-error --fail --max-time {reuse_timeout_s} "http://127.0.0.1:{debug_port}/json/version" >/dev/null 2>&1; then
		echo "browser_reused=true debug_port={debug_port}"
		exit 0
	fi
fi
BROWSER_BIN=""
for candidate in {candidates}; do
	if command -v "$candidate" >/dev/null 2>&1; then
		BROWSER_BIN="$candidate"
		break
	fi
done
if [ -z "$BROWSER_BIN" ]; then
	echo "no Chrome-compatible browser binary found in PATH (tried: {candidate_log})" >&2
	exit 127
fi
mkdir -p {profile_dir}
mkdir -p "$(dirname {log_path})"
"$BROWSER_BIN" {launch_flags} --remote-debugging-address=127.0.0.1 --remote-debugging-port={debug_port} --no-first-run --no-default-browser-check --disable-dev-shm-usage --no-sandbox --user-data-dir={profile_dir} about:blank >{log_path} 2>&1 &
echo "browser_binary=$BROWSER_BIN debug_port={debug_port}""#,
        reuse_timeout_s = BROWSER_REUSE_MAX_TIME_SECONDS,
        candidates = shell_quote_all(&BROWSER_CANDIDATES),
        candidate_log = BROWSER_CANDIDATES.join(", "),
        launch_flags = display_mode.launch_flags(),
        debug_port = debug_port,
        profile_dir = shell_quote(profile_dir),
        log_path = shell_quote(log_path),
    )
}

#[derive(Clone, Copy)]
enum BrowserDisplayMode {
    Background,
    Foreground,
}

impl BrowserDisplayMode {
    fn from_raw(raw: &str) -> Result<Self, String> {
        match raw.trim().to_ascii_lowercase().as_str() {
            BROWSER_DISPLAY_MODE_BACKGROUND => Ok(Self::Background),
            BROWSER_DISPLAY_MODE_FOREGROUND => Ok(Self::Foreground),
            _ => Err(format!(
                "display_mode must be one of: {}, {}",
                BROWSER_DISPLAY_MODE_BACKGROUND, BROWSER_DISPLAY_MODE_FOREGROUND
            )),
        }
    }

    fn launch_flags(self) -> &'static str {
        match self {
            Self::Background => "--headless --disable-gpu",
            Self::Foreground => "--disable-gpu",
        }
    }
}

fn shell_quote_all(values: &[&str]) -> String {
    values
        .iter()
        .map(|value| shell_quote(value))
        .collect::<Vec<String>>()
        .join(" ")
}

fn shell_quote(raw: &str) -> String {
    let escaped = raw.replace('\'', "'\"'\"'");
    format!("'{escaped}'")
}

#[cfg(test)]
mod tests {
    use super::{
        browser_launch_command, dispatch_action, handle_browser_launch, MAX_BROWSER_DEBUG_PORT,
    };
    use serde_json::json;

    #[test]
    fn dispatch_action_routes_browser_launch() {
        let response = dispatch_action(
            "BROWSER_LAUNCH",
            &json!({"command": shell_echo_command("ok")}),
        );
        let response = response.expect("BROWSER_LAUNCH should be routed");
        assert_eq!(response.status, "success");
        assert!(response.payload["stdout"]
            .as_str()
            .unwrap_or_default()
            .contains("ok"));
    }

    #[test]
    fn handle_browser_launch_rejects_invalid_debug_port() {
        let response = handle_browser_launch(&json!({
            "command": shell_echo_command("ok"),
            "debug_port": MAX_BROWSER_DEBUG_PORT + 1
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("debug_port must be between"));
    }

    #[test]
    fn browser_launch_command_builds_auto_launch_script() {
        let command = browser_launch_command(&json!({"debug_port": 9333})).expect("must build");
        assert!(command.contains("/json/version"));
        assert!(command.contains("google-chrome"));
        assert!(command.contains("--remote-debugging-port=9333"));
        assert!(command.contains("--headless"));
        assert!(command.contains("/tmp/ghost-browser-control-profile/port-9333"));
        assert!(command.contains("/tmp/ghost-browser-control-log/port-9333.log"));
    }

    #[test]
    fn browser_launch_command_builds_foreground_auto_launch_script() {
        let command =
            browser_launch_command(&json!({"debug_port": 9444, "display_mode": "foreground"}))
                .expect("must build");
        assert!(command.contains("--remote-debugging-port=9444"));
        assert!(!command.contains("--headless"));
    }

    #[test]
    fn browser_launch_command_uses_session_profile_key_when_provided() {
        let command = browser_launch_command(&json!({
            "debug_port": 9555,
            "session_id": "session A/1"
        }))
        .expect("must build");
        assert!(command.contains("/tmp/ghost-browser-control-profile/session-session_A_1"));
        assert!(command.contains("/tmp/ghost-browser-control-log/session-session_A_1.log"));
    }

    #[test]
    fn browser_launch_command_rejects_invalid_display_mode() {
        let err = browser_launch_command(&json!({"display_mode": "invalid"}))
            .expect_err("invalid display_mode should fail");
        assert!(err.contains("display_mode must be one of"));
    }

    #[test]
    fn dispatch_action_returns_none_for_non_browser_action() {
        assert!(dispatch_action("SCRIPT_EXEC", &json!({})).is_none());
    }

    fn shell_echo_command(text: &str) -> String {
        if cfg!(target_os = "windows") {
            format!("echo {text}")
        } else {
            format!("printf '{text}'")
        }
    }
}
