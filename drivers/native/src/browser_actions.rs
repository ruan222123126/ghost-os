use crate::Response;
use crate::json_params::{optional_string, optional_usize};
use crate::sandbox::SandboxConfig;
use crate::sandbox::shell_tools::{
    bash_exec_impl, format_bash_exec_failure, resolve_shell_timeout,
};
use serde_json::{Value, json};

const DEFAULT_BROWSER_DEBUG_PORT: usize = 9222;
const MIN_BROWSER_DEBUG_PORT: usize = 1;
const MAX_BROWSER_DEBUG_PORT: usize = 65535;
const DEFAULT_BROWSER_PROFILE_DIR: &str = "/tmp/ghost-browser-control-profile";
const DEFAULT_BROWSER_LOG_PATH: &str = "/tmp/ghost-browser-control.log";

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
    let debug_port = optional_debug_port(params)?;
    if let Some(command) = optional_string(params, "command")? {
        return Ok(command);
    }
    let launch_port = debug_port.unwrap_or(DEFAULT_BROWSER_DEBUG_PORT);
    Ok(build_auto_browser_launch_command(launch_port))
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

fn build_auto_browser_launch_command(debug_port: usize) -> String {
    format!(
        r#"set -euo pipefail
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
"$BROWSER_BIN" --headless --disable-gpu --remote-debugging-address=127.0.0.1 --remote-debugging-port={debug_port} --no-first-run --no-default-browser-check --disable-dev-shm-usage --no-sandbox --user-data-dir={profile_dir} about:blank >{log_path} 2>&1 &
echo "browser_binary=$BROWSER_BIN""#,
        candidates = shell_quote_all(&BROWSER_CANDIDATES),
        candidate_log = BROWSER_CANDIDATES.join(", "),
        debug_port = debug_port,
        profile_dir = shell_quote(DEFAULT_BROWSER_PROFILE_DIR),
        log_path = shell_quote(DEFAULT_BROWSER_LOG_PATH),
    )
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
        MAX_BROWSER_DEBUG_PORT, browser_launch_command, dispatch_action, handle_browser_launch,
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
        assert!(
            response.payload["stdout"]
                .as_str()
                .unwrap_or_default()
                .contains("ok")
        );
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
        assert!(command.contains("google-chrome"));
        assert!(command.contains("--remote-debugging-port=9333"));
        assert!(command.contains("/tmp/ghost-browser-control-profile"));
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
