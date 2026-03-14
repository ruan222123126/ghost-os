use crate::Response;
use crate::json_params::{optional_usize, required_string};
use crate::sandbox::SandboxConfig;
use crate::sandbox::shell_tools::{
    bash_exec_impl, format_bash_exec_failure, resolve_shell_timeout,
};
use serde_json::{Value, json};

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "BASH_EXEC" => Some(handle_bash_exec(params)),
        _ => None,
    }
}

fn handle_bash_exec(params: &Value) -> Response {
    let command = match required_string(params, "command") {
        Ok(command) => command,
        Err(err) => return Response::error(err),
    };

    let config = SandboxConfig::default();
    let timeout_ms = match optional_usize(params, "timeout_ms") {
        Ok(timeout_ms) => resolve_shell_timeout(&config, timeout_ms.map(|value| value as u64)),
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
