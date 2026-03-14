use serde_json::{Value, json};

use crate::{Response, codex_cli, file_actions, input, screen, script_exec, shell_actions};

pub(crate) struct PersistentState {
    codex_manager: codex_cli::CodexCommandManager,
}

impl PersistentState {
    pub(crate) fn new() -> Self {
        Self {
            codex_manager: codex_cli::CodexCommandManager::new(),
        }
    }
}

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

    if let Some(response) = input::dispatch_action(action, params) {
        return response;
    }

    if let Some(response) = codex_cli::dispatch_action(action, params) {
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

// dispatch_with_state 为 persistent 模式预留轻量级状态扩展点。
pub(crate) fn dispatch_with_state(
    action: &str,
    params: &Value,
    trace_id: &str,
    state: &mut PersistentState,
) -> Response {
    if let Some(response) = codex_cli::dispatch_with_state(action, params, &mut state.codex_manager)
    {
        return response;
    }
    dispatch_action(action, params, trace_id)
}
