use crate::Response;
use crate::json_params::{optional_bool, optional_string, optional_usize, required_string};
use crate::sandbox::SandboxConfig;
use crate::sandbox::shell_tools::{
    bash_exec_impl, format_bash_exec_failure, resolve_shell_output_chars, resolve_shell_timeout,
};
use crate::shell_sessions::{ShellSessionRequest, run_shell_session};
use serde_json::{Value, json};

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "BASH_EXEC" => Some(handle_bash_exec(params)),
        _ => None,
    }
}

struct BashExecRequest {
    command: String,
    login: bool,
    interactive: bool,
    session_id: Option<String>,
    tty: bool,
    yield_time_ms: Option<u64>,
    timeout_ms: Option<u64>,
    max_output_chars: Option<usize>,
}

fn handle_bash_exec(params: &Value) -> Response {
    let request = match parse_bash_exec_request(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    if request.interactive {
        return handle_interactive_bash_exec(request);
    }
    handle_oneshot_bash_exec(request)
}

fn parse_bash_exec_request(params: &Value) -> Result<BashExecRequest, String> {
    let command = required_string(params, "command")?;
    let interactive = optional_bool(params, "interactive")?.unwrap_or(false);
    let login = optional_bool(params, "login")?.unwrap_or(true);
    let session_id = optional_string(params, "session_id")?;
    let tty = optional_bool(params, "tty")?.unwrap_or(false);
    let yield_time_ms = optional_u64_param(params, "yield_time_ms")?;
    let timeout_ms = optional_u64_param(params, "timeout_ms")?;
    let max_output_chars = optional_usize(params, "max_output_chars")?;
    validate_bash_exec_request(params, interactive, &session_id)?;
    Ok(BashExecRequest {
        command,
        login,
        interactive,
        session_id,
        tty,
        yield_time_ms,
        timeout_ms,
        max_output_chars,
    })
}

fn optional_u64_param(params: &Value, field: &str) -> Result<Option<u64>, String> {
    let value = optional_usize(params, field)?;
    value
        .map(|item| u64::try_from(item).map_err(|_| format!("{field} is too large")))
        .transpose()
}

fn validate_bash_exec_request(
    params: &Value,
    interactive: bool,
    session_id: &Option<String>,
) -> Result<(), String> {
    let has_login = params.get("login").is_some();
    let has_timeout = params.get("timeout_ms").is_some();
    let has_tty = params.get("tty").is_some();
    let has_yield = params.get("yield_time_ms").is_some();

    if interactive {
        if has_login {
            return Err("login is only allowed when interactive=false".to_string());
        }
        if has_timeout {
            return Err("timeout_ms is only allowed when interactive=false".to_string());
        }
        return Ok(());
    }

    if session_id.is_some() {
        return Err("session_id is only allowed when interactive=true".to_string());
    }
    if has_tty || has_yield {
        return Err("tty and yield_time_ms are only allowed when interactive=true".to_string());
    }
    Ok(())
}

fn handle_oneshot_bash_exec(request: BashExecRequest) -> Response {
    let config = SandboxConfig::default();
    let timeout_ms = resolve_shell_timeout(&config, request.timeout_ms);
    let max_output_chars = match resolve_shell_output_chars(&config, request.max_output_chars) {
        Ok(limit) => Some(limit),
        Err(err) => return Response::error(err),
    };

    let result = match bash_exec_impl(
        &config,
        &request.command,
        request.login,
        Some(timeout_ms),
        max_output_chars,
    ) {
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

fn handle_interactive_bash_exec(request: BashExecRequest) -> Response {
    let config = SandboxConfig::default();
    let max_output_chars = match resolve_shell_output_chars(&config, request.max_output_chars) {
        Ok(limit) => limit,
        Err(err) => return Response::error(err),
    };

    let response = match run_shell_session(ShellSessionRequest {
        command: request.command,
        session_id: request.session_id,
        tty: request.tty,
        yield_time_ms: request.yield_time_ms,
        max_output_chars,
    }) {
        Ok(response) => response,
        Err(err) => return Response::error(err),
    };

    eprintln!(
        "native checkpoint component=shell interactive=true session_id={} reused={} running={}",
        response.session_id, response.reused, response.running
    );

    if let Some(code) = response.command_exit_code
        && code != 0
    {
        return Response::error(format!("command failed: exit status {code}"));
    }

    Response::success(json!({
        "session_id": response.session_id,
        "stdout": response.stdout,
        "stderr": response.stderr,
        "running": response.running,
        "exit_code": response.shell_exit_code,
    }))
}

#[cfg(test)]
mod tests {
    use super::{dispatch_action, handle_bash_exec};
    use serde_json::json;

    #[test]
    fn handle_bash_exec_returns_error_when_command_is_missing() {
        let response = handle_bash_exec(&json!({}));
        assert_eq!(response.status, "error");
        assert_eq!(response.error, "command is required");
    }

    #[test]
    fn handle_bash_exec_returns_output() {
        let response = handle_bash_exec(&json!({"command": shell_echo_command("hi")}));
        assert_eq!(response.status, "success");
        let stdout = response.payload["stdout"]
            .as_str()
            .expect("stdout should be a string");
        assert!(stdout.contains("hi"), "unexpected stdout: {stdout}");
    }

    #[test]
    fn handle_bash_exec_login_false_uses_non_login_shell() {
        let response = handle_bash_exec(&json!({
            "command": shell_echo_command("ok"),
            "login": false
        }));
        assert_eq!(response.status, "success");
    }

    #[test]
    fn handle_bash_exec_rejects_zero_output_limit() {
        let response = handle_bash_exec(&json!({
            "command": shell_echo_command("hi"),
            "max_output_chars": 0
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("max_output_chars"));
    }

    #[test]
    fn handle_bash_exec_rejects_invalid_param_combinations() {
        let cases = vec![
            json!({"command":"pwd","session_id":"sess-1"}),
            json!({"command":"pwd","interactive":true,"login":true}),
            json!({"command":"pwd","interactive":true,"timeout_ms":100}),
            json!({"command":"pwd","tty":true}),
        ];
        for params in cases {
            let response = handle_bash_exec(&params);
            assert_eq!(response.status, "error");
        }
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_times_out() {
        let response = handle_bash_exec(&json!({
            "command": "sleep 1",
            "timeout_ms": 10
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("timed out"));
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_interactive_creates_and_reuses_session() {
        let create = handle_bash_exec(&json!({
            "command":"printf shell-interactive",
            "interactive":true,
            "yield_time_ms":300
        }));
        assert_eq!(create.status, "success");
        let session_id = create.payload["session_id"]
            .as_str()
            .expect("session_id should be string")
            .to_string();
        assert!(!session_id.is_empty(), "session_id should not be empty");

        let reuse = handle_bash_exec(&json!({
            "command":"printf shell-reuse",
            "interactive":true,
            "session_id":session_id,
            "yield_time_ms":300
        }));
        assert_eq!(reuse.status, "success");
        let stdout = reuse.payload["stdout"]
            .as_str()
            .expect("stdout should be string");
        assert!(
            stdout.contains("shell-reuse"),
            "unexpected stdout: {stdout}"
        );
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_interactive_command_failure_returns_error() {
        let response = handle_bash_exec(&json!({
            "command":"false",
            "interactive":true,
            "yield_time_ms":300
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("exit status"));
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_interactive_rejects_tty() {
        let response = handle_bash_exec(&json!({
            "command":"echo hi",
            "interactive":true,
            "tty":true
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("tty=true"));
    }

    #[cfg(not(target_os = "windows"))]
    #[test]
    fn handle_bash_exec_interactive_unknown_session_errors() {
        let response = handle_bash_exec(&json!({
            "command":"echo hi",
            "interactive":true,
            "session_id":"missing-session"
        }));
        assert_eq!(response.status, "error");
        assert!(response.error.contains("unknown session_id"));
    }

    #[test]
    fn dispatch_action_returns_none_for_unknown_shell_action() {
        assert!(dispatch_action("LIST_FILES", &json!({})).is_none());
    }

    fn shell_echo_command(text: &str) -> String {
        if cfg!(target_os = "windows") {
            format!("echo {text}")
        } else {
            format!("printf '{text}'")
        }
    }
}
