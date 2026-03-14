mod command;
mod manager;
mod output;
mod params;
mod start;
mod status;

#[cfg(test)]
mod tests;

use serde_json::Value;

use crate::Response;

pub(crate) use manager::CodexCommandManager;

const DEFAULT_WAIT_MS_BEFORE_ASYNC: u64 = 3000;
const DEFAULT_WAIT_DURATION_SECONDS: u64 = 300;
const DEFAULT_OUTPUT_CHAR_COUNT: usize = 200;
const MAX_OUTPUT_BUFFER_CHARS: usize = 4000;
const STATUS_POLL_INTERVAL_MS: u64 = 500;
const UNKNOWN_EXIT_CODE: i32 = -1;

pub(crate) fn dispatch_action(action: &str, _params: &Value) -> Option<Response> {
    match action {
        "CODEX_CLI_START" | "CODEX_CLI_STATUS" => Some(Response::error(
            "codex_cli requires native_persistent=true".to_string(),
        )),
        _ => None,
    }
}

pub(crate) fn dispatch_with_state(
    action: &str,
    params: &Value,
    state: &mut CodexCommandManager,
) -> Option<Response> {
    match action {
        "CODEX_CLI_START" => Some(start::handle_start(params, state)),
        "CODEX_CLI_STATUS" => Some(status::handle_status(params, state)),
        _ => None,
    }
}
