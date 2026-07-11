#[cfg(feature = "python-sandbox")]
mod budget;
mod result;
#[cfg(feature = "python-sandbox")]
mod types;
#[cfg(feature = "python-sandbox")]
mod worker;

use serde_json::Value;

#[cfg(feature = "python-sandbox")]
use serde_json::json;

use crate::Response;
#[cfg(feature = "python-sandbox")]
use crate::read_stdin_payload;
#[cfg(feature = "python-sandbox")]
use crate::sandbox::{PythonSandbox, SandboxConfig};

#[cfg(feature = "python-sandbox")]
use budget::resolve_script_budget;
use result::emit_worker_result;
#[cfg(feature = "python-sandbox")]
use types::ScriptWorkerRequest;
#[cfg(feature = "python-sandbox")]
use worker::{apply_memory_limit, execute_script_in_subprocess};

#[cfg(not(feature = "python-sandbox"))]
const PYTHON_SANDBOX_DISABLED_ERROR: &str =
    "SCRIPT_EXEC unavailable: native build does not include `python-sandbox` feature";

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "SCRIPT_EXEC" => Some(handle_script_exec(params)),
        _ => None,
    }
}

#[cfg(feature = "python-sandbox")]
pub(crate) fn run_sandbox_worker() {
    let request = match read_worker_request() {
        Ok(request) => request,
        Err(err) => return emit_execution_error(err),
    };

    if let Err(err) = apply_memory_limit(request.max_memory_mb) {
        return emit_execution_error(err);
    }

    let sandbox = PythonSandbox::new(request.sandbox_config);
    emit_worker_result(sandbox.execute_blocking(&request.script));
}

#[cfg(not(feature = "python-sandbox"))]
pub(crate) fn run_sandbox_worker() {
    emit_execution_error(PYTHON_SANDBOX_DISABLED_ERROR.to_string());
}

#[cfg(feature = "python-sandbox")]
fn handle_script_exec(params: &Value) -> Response {
    let script = match parse_script(params) {
        Ok(script) => script,
        Err(err) => return Response::error(err),
    };

    let sandbox_config = SandboxConfig::default();
    let budget = match resolve_script_budget(params, &sandbox_config) {
        Ok(budget) => budget,
        Err(err) => return Response::error(err),
    };

    let result = match execute_script_in_subprocess(&script, budget, sandbox_config) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    match result.error {
        Some(err) => Response::error(err),
        None => Response::success(json!({
            "output": result.output,
            "tool_calls_log": result.tool_calls_log,
        })),
    }
}

#[cfg(not(feature = "python-sandbox"))]
fn handle_script_exec(_params: &Value) -> Response {
    Response::error(PYTHON_SANDBOX_DISABLED_ERROR.to_string())
}

#[cfg(feature = "python-sandbox")]
fn read_worker_request() -> Result<ScriptWorkerRequest, String> {
    let input = read_stdin_payload()?;
    serde_json::from_str(&input).map_err(|err| format!("invalid sandbox worker request: {err}"))
}

#[cfg(feature = "python-sandbox")]
fn parse_script(params: &Value) -> Result<String, String> {
    let script = params
        .get("script")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");
    if script.is_empty() {
        return Err("script is required".to_string());
    }
    Ok(script.to_string())
}

fn emit_execution_error(err: String) {
    emit_worker_result(crate::sandbox::ExecutionResult {
        output: String::new(),
        tool_calls_log: Vec::new(),
        error: Some(err),
    });
}

#[cfg(test)]
mod tests {
    use super::dispatch_action;
    use serde_json::json;

    #[test]
    fn dispatch_action_returns_none_for_unknown_script_action() {
        assert!(dispatch_action("BASH_EXEC", &json!({})).is_none());
    }

    #[cfg(not(feature = "python-sandbox"))]
    #[test]
    fn dispatch_action_reports_missing_python_sandbox_feature() {
        let response = dispatch_action("SCRIPT_EXEC", &json!({}))
            .expect("SCRIPT_EXEC should be handled with explicit error");
        assert_eq!(response.status, "error");
        assert!(response.error.contains("python-sandbox"));
    }
}
