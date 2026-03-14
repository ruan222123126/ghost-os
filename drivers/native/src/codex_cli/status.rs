use std::thread;
use std::time::{Duration, Instant};

use serde_json::{Value, json};

use crate::Response;

use super::STATUS_POLL_INTERVAL_MS;
use super::command::CodexCommand;
use super::manager::CodexCommandManager;
use super::params::StatusParams;

pub(super) fn handle_status(params: &Value, state: &mut CodexCommandManager) -> Response {
    let request = match StatusParams::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let Some(command) = find_command(state, &request) else {
        return command_not_found_response();
    };
    poll_command_status(
        command,
        request.wait_duration_seconds,
        request.output_char_count,
    )
}

fn find_command<'a>(
    state: &'a mut CodexCommandManager,
    request: &StatusParams,
) -> Option<&'a mut CodexCommand> {
    if request.has_command_id() {
        if state.contains(&request.command_id) {
            return state.get_mut(&request.command_id);
        }
        return state.find_by_session_id(&request.session_id);
    }
    if request.has_session_id() {
        return state.find_by_session_id(&request.session_id);
    }
    None
}

fn poll_command_status(
    command: &mut CodexCommand,
    wait_duration_seconds: u64,
    output_char_count: usize,
) -> Response {
    let deadline = Instant::now() + Duration::from_secs(wait_duration_seconds);
    loop {
        match command.update_exit_code() {
            Ok(Some(code)) => {
                return snapshot_response("done", command, output_char_count, Some(code));
            }
            Ok(None) => {}
            Err(err) => {
                return command_error_response("error", command, output_char_count, err);
            }
        }
        if Instant::now() >= deadline {
            return snapshot_response("running", command, output_char_count, None);
        }
        thread::sleep(Duration::from_millis(STATUS_POLL_INTERVAL_MS));
    }
}

fn command_not_found_response() -> Response {
    Response::success(json!({
        "status": "error",
        "message": "command not found",
    }))
}

fn snapshot_response(
    status: &str,
    command: &CodexCommand,
    output_char_count: usize,
    exit_code: Option<i32>,
) -> Response {
    Response::success(json!({
        "status": status,
        "command_id": command.id(),
        "session_id": command.session_id_value().unwrap_or_default(),
        "exit_code": exit_code,
        "output_tail": command.output_tail(output_char_count),
        "output_path": command.output_path_value(),
    }))
}

fn command_error_response(
    status: &str,
    command: &CodexCommand,
    output_char_count: usize,
    message: String,
) -> Response {
    Response::success(json!({
        "status": status,
        "command_id": command.id(),
        "session_id": command.session_id_value().unwrap_or_default(),
        "output_tail": command.output_tail(output_char_count),
        "output_path": command.output_path_value(),
        "message": message,
    }))
}
