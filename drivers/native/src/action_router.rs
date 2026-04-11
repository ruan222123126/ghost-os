use serde_json::{Value, json};

use crate::{Response, codex_cli, file_actions, input, screen, script_exec, shell_actions};

// dispatch_action 统一处理协议 action 到具体原子能力的映射。
pub(crate) fn dispatch_action(action: &str, params: &Value, trace_id: &str) -> Response {
    if action == "PING" {
        return Response::success(json!({
            "message": "PONG",
            "trace_id": trace_id
        }));
    }

    if let Some(response) = file_actions::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = shell_actions::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = codex_cli::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = input::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = screen::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = script_exec::dispatch_action(action, params) {
        return response;
    }

    Response::error(format!("unsupported action: {action}"))
}
