use std::process::{Child, Command, Stdio};
use std::thread;
use std::time::Duration;

use serde_json::{Value, json};

use crate::Response;

use super::command::CodexCommand;
use super::manager::CodexCommandManager;
use super::output::{
    SharedOutputBuffer, SharedSessionId, new_session_id_slot, new_shared_output,
    spawn_output_reader,
};
use super::params::StartParams;

struct SpawnedCodexProcess {
    child: Child,
    output: SharedOutputBuffer,
    session_id: SharedSessionId,
    output_path: Option<String>,
}

pub(super) fn handle_start(params: &Value, state: &mut CodexCommandManager) -> Response {
    let request = match StartParams::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let spawned = match spawn_codex_process(&request) {
        Ok(spawned) => spawned,
        Err(err) => return spawn_error_response(err),
    };
    let (seq, command_id) = state.next_command_id();
    let mut command = CodexCommand::new(
        seq,
        command_id.clone(),
        spawned.child,
        spawned.output,
        spawned.session_id,
        spawned.output_path,
    );
    wait_before_async(request.wait_ms_before_async);
    let response = match start_response(&command_id, &mut command, request.output_char_count) {
        Ok(response) => response,
        Err(response) => return response,
    };
    state.insert(command);
    response
}

fn spawn_codex_process(request: &StartParams) -> Result<SpawnedCodexProcess, String> {
    let mut command = build_process_command(request);
    let mut child = command
        .spawn()
        .map_err(|err| format!("spawn codex failed: {err}"))?;
    let stdout = child.stdout.take();
    let stderr = child.stderr.take();
    let output = new_shared_output();
    let session_id = new_session_id_slot();
    spawn_output_reader(stdout, output.clone(), session_id.clone(), "STDOUT: ");
    spawn_output_reader(stderr, output.clone(), session_id.clone(), "STDERR: ");
    Ok(SpawnedCodexProcess {
        child,
        output,
        session_id,
        output_path: request.output_path_value(),
    })
}

fn build_process_command(request: &StartParams) -> Command {
    let mut command = Command::new("codex");
    command.stdout(Stdio::piped()).stderr(Stdio::piped());
    if let Some(dir) = request.cwd.as_ref() {
        command.current_dir(dir);
    }
    request.operation.apply(&mut command, request.prompt.trim());
    apply_common_flags(&mut command, request);
    command
}

fn apply_common_flags(command: &mut Command, request: &StartParams) {
    if request.full_auto {
        command.arg("--full-auto");
    }
    if request.skip_git_repo_check {
        command.arg("--skip-git-repo-check");
    }
    if request.json_flag {
        command.arg("--json");
    }
    if let Some(model) = request.model.as_ref()
        && !model.trim().is_empty()
    {
        command.arg("-m").arg(model.trim());
    }
    if let Some(dir) = request.cwd.as_ref() {
        command.arg("-C").arg(dir);
    }
    if let Some(path) = request.output_path.as_ref() {
        command.arg("-o").arg(path);
    }
}

fn wait_before_async(wait_ms_before_async: u64) {
    if wait_ms_before_async > 0 {
        thread::sleep(Duration::from_millis(wait_ms_before_async));
    }
}

fn start_response(
    command_id: &str,
    command: &mut CodexCommand,
    output_char_count: usize,
) -> Result<Response, Response> {
    match command.update_exit_code() {
        Ok(Some(code)) => Ok(snapshot_response(
            "done",
            command_id,
            command,
            output_char_count,
            Some(code),
        )),
        Ok(None) => Ok(snapshot_response(
            "running",
            command_id,
            command,
            output_char_count,
            None,
        )),
        Err(err) => Err(command_error_response(
            "error",
            command_id,
            command,
            output_char_count,
            err,
        )),
    }
}

fn spawn_error_response(message: String) -> Response {
    Response::success(json!({
        "status": "error",
        "message": message,
    }))
}

fn snapshot_response(
    status: &str,
    command_id: &str,
    command: &CodexCommand,
    output_char_count: usize,
    exit_code: Option<i32>,
) -> Response {
    Response::success(json!({
        "status": status,
        "command_id": command_id,
        "session_id": command.session_id_value().unwrap_or_default(),
        "exit_code": exit_code,
        "output_tail": command.output_tail(output_char_count),
        "output_path": command.output_path_value(),
    }))
}

fn command_error_response(
    status: &str,
    command_id: &str,
    command: &CodexCommand,
    output_char_count: usize,
    message: String,
) -> Response {
    Response::success(json!({
        "status": status,
        "command_id": command_id,
        "session_id": command.session_id_value().unwrap_or_default(),
        "output_tail": command.output_tail(output_char_count),
        "output_path": command.output_path_value(),
        "message": message,
    }))
}
