use serde_json::{Value, json};
use std::thread;
use std::time::{Duration, Instant};

use crate::Response;
use crate::json_params::{optional_bool, optional_string, optional_usize, required_string};

use super::{
    CODEX_CLI_STATUS_POLL_INTERVAL_MS, DEFAULT_OUTPUT_CHARACTER_COUNT,
    DEFAULT_WAIT_DURATION_SECONDS, DEFAULT_WAIT_MS_BEFORE_ASYNC, StartRequest, StatusRequest,
    worker,
};

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "CODEX_CLI_START" => Some(handle_start(params)),
        "CODEX_CLI_STATUS" => Some(handle_status(params)),
        _ => None,
    }
}

fn handle_start(params: &Value) -> Response {
    let request = match parse_start_request(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let output_path = request
        .output_path
        .clone()
        .unwrap_or_else(worker::default_output_path);
    let exit_code_path = format!("{output_path}.exit");
    let args = match build_codex_cli_args(&request, &output_path) {
        Ok(args) => args,
        Err(err) => return Response::error(err),
    };
    if let Err(err) = worker::spawn_worker_process(worker::WorkerSpawnInput {
        args,
        working_dir: request.working_dir.clone(),
        output_path: output_path.clone(),
        exit_code_path: exit_code_path.clone(),
        codex_executable_path: request.codex_executable_path.clone(),
        node_executable_path: request.node_executable_path.clone(),
    }) {
        return Response::error(err);
    }
    wait_before_async(request.wait_ms_before_async);
    let output_tail =
        worker::read_output_tail(&output_path, request.output_character_count).unwrap_or_default();
    match worker::read_exit_code(&exit_code_path) {
        Ok(Some(exit_code)) => finished_response(
            json!({
                "output_path": output_path,
                "exit_code_path": exit_code_path,
                "output_tail": output_tail,
                "exit_code": exit_code
            }),
            &output_path,
        ),
        Ok(None) => Response::success(json!({
            "output_path": output_path,
            "exit_code_path": exit_code_path,
            "output_tail": output_tail
        })),
        Err(err) => Response::error(err),
    }
}

fn handle_status(params: &Value) -> Response {
    let request = match parse_status_request(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let deadline = Instant::now() + Duration::from_secs(request.wait_duration_seconds as u64);
    loop {
        let output_tail =
            match worker::read_output_tail(&request.output_path, request.output_character_count) {
                Ok(output) => output,
                Err(err) => return Response::error(err),
            };
        let exit_code = match worker::read_exit_code(&request.exit_code_path) {
            Ok(exit_code) => exit_code,
            Err(err) => return Response::error(err),
        };
        if let Some(exit_code) = exit_code {
            return finished_response(
                json!({"output_tail": output_tail, "exit_code": exit_code}),
                &request.output_path,
            );
        }
        if Instant::now() >= deadline {
            return Response::success(json!({"output_tail": output_tail}));
        }
        thread::sleep(Duration::from_millis(CODEX_CLI_STATUS_POLL_INTERVAL_MS));
    }
}

fn finished_response(payload: Value, output_path: &str) -> Response {
    match with_final_agent_message(payload, output_path) {
        Ok(enriched) => Response::success(enriched),
        Err(err) => Response::error(err),
    }
}

fn with_final_agent_message(payload: Value, output_path: &str) -> Result<Value, String> {
    let mut enriched = payload;
    if let Some(message) = worker::read_final_agent_message(output_path)? {
        enriched["final_message"] = json!(message);
    }
    Ok(enriched)
}

fn parse_start_request(params: &Value) -> Result<StartRequest, String> {
    let wait_ms_before_async =
        optional_usize(params, "wait_ms_before_async")?.unwrap_or(DEFAULT_WAIT_MS_BEFORE_ASYNC);
    let output_character_count =
        optional_usize(params, "output_character_count")?.unwrap_or(DEFAULT_OUTPUT_CHARACTER_COUNT);
    let request = StartRequest {
        op: required_string(params, "op")?,
        prompt: required_string(params, "prompt")?,
        session_id: optional_string(params, "session_id")?,
        working_dir: required_string(params, "working_dir")?,
        use_cwd_flag: optional_bool(params, "use_cwd_flag")?.unwrap_or(false),
        output_path: optional_string(params, "output_path")?,
        model: optional_string(params, "model")?,
        codex_executable_path: optional_string(params, "codex_executable_path")?,
        node_executable_path: optional_string(params, "node_executable_path")?,
        sandbox: parse_sandbox(params)?,
        full_auto: optional_bool(params, "full_auto")?,
        skip_git_repo_check: optional_bool(params, "skip_git_repo_check")?.unwrap_or(true),
        json_flag: optional_bool(params, "json")?.unwrap_or(true),
        wait_ms_before_async,
        output_character_count,
    };
    validate_start_request(&request)?;
    Ok(request)
}

fn validate_start_request(request: &StartRequest) -> Result<(), String> {
    if request.sandbox.is_some() && request.full_auto.is_some() {
        return Err("sandbox and full_auto cannot be used together".to_string());
    }
    Ok(())
}

fn parse_sandbox(params: &Value) -> Result<Option<String>, String> {
    let sandbox = optional_string(params, "sandbox")?;
    let Some(value) = sandbox else {
        return Ok(None);
    };
    if is_supported_sandbox(&value) {
        return Ok(Some(value));
    }
    Err(format!("unsupported sandbox: {value}"))
}

fn is_supported_sandbox(value: &str) -> bool {
    matches!(
        value,
        "read-only" | "workspace-write" | "danger-full-access"
    )
}

fn parse_status_request(params: &Value) -> Result<StatusRequest, String> {
    Ok(StatusRequest {
        output_path: required_string(params, "output_path")?,
        exit_code_path: required_string(params, "exit_code_path")?,
        wait_duration_seconds: optional_usize(params, "wait_duration_seconds")?
            .unwrap_or(DEFAULT_WAIT_DURATION_SECONDS),
        output_character_count: optional_usize(params, "output_character_count")?
            .unwrap_or(DEFAULT_OUTPUT_CHARACTER_COUNT),
    })
}

fn build_codex_cli_args(request: &StartRequest, output_path: &str) -> Result<Vec<String>, String> {
    let mut args = operation_args(request)?;
    append_sandbox_args(&mut args, request);
    if request.skip_git_repo_check {
        args.push("--skip-git-repo-check".to_string());
    }
    if request.json_flag {
        args.push("--json".to_string());
    }
    if let Some(model) = request.model.as_ref() {
        args.push("-m".to_string());
        args.push(model.clone());
    }
    if request.use_cwd_flag {
        args.push("-C".to_string());
        args.push(request.working_dir.clone());
    }
    if request.output_path.is_some() {
        args.push("-o".to_string());
        args.push(output_path.to_string());
    }
    Ok(args)
}

fn append_sandbox_args(args: &mut Vec<String>, request: &StartRequest) {
    let sandbox = request
        .sandbox
        .as_deref()
        .or_else(|| legacy_full_auto_sandbox(request.full_auto));
    if let Some(value) = sandbox {
        args.push("--sandbox".to_string());
        args.push(value.to_string());
    }
}

fn legacy_full_auto_sandbox(full_auto: Option<bool>) -> Option<&'static str> {
    match full_auto {
        Some(true) => Some("workspace-write"),
        _ => None,
    }
}

fn operation_args(request: &StartRequest) -> Result<Vec<String>, String> {
    match request.op.as_str() {
        "start" => Ok(vec!["exec".to_string(), request.prompt.clone()]),
        "resume" => {
            let session_id = request
                .session_id
                .clone()
                .ok_or_else(|| "session_id is required for resume".to_string())?;
            Ok(vec![
                "exec".to_string(),
                "resume".to_string(),
                session_id,
                request.prompt.clone(),
            ])
        }
        "fork" => Err("fork is interactive-only in Codex CLI 0.130.0".to_string()),
        _ => Err(format!("unsupported op: {}", request.op)),
    }
}

fn wait_before_async(wait_ms: usize) {
    if wait_ms == 0 {
        return;
    }
    thread::sleep(Duration::from_millis(wait_ms as u64));
}

#[cfg(test)]
mod tests {
    use super::dispatch_action;
    use serde_json::json;
    use std::fs;
    use std::path::PathBuf;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn dispatch_status_returns_exit_code_when_worker_finished() {
        let dir = temp_dir_path("status-finished");
        fs::create_dir_all(&dir).expect("temp dir should be created");
        let output_path = dir.join("output.log");
        let exit_code_path = dir.join("output.log.exit");
        fs::write(&output_path, "line-one\nline-two").expect("output file should be written");
        fs::write(&exit_code_path, "0").expect("exit file should be written");

        let response = dispatch_action(
            "CODEX_CLI_STATUS",
            &json!({
                "output_path": output_path.to_string_lossy(),
                "exit_code_path": exit_code_path.to_string_lossy(),
                "wait_duration_seconds": 1,
                "output_character_count": 8
            }),
        )
        .expect("status action must be handled");

        assert_eq!(response.status, "success");
        assert_eq!(response.payload["exit_code"], 0);
        assert_eq!(response.payload["output_tail"], "line-two");
    }

    fn temp_dir_path(prefix: &str) -> PathBuf {
        let stamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos();
        let mut path = std::env::temp_dir();
        path.push(format!("ghost-native-codex-{prefix}-{stamp}"));
        path
    }
}
