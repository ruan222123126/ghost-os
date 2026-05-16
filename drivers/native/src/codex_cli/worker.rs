use serde_json::Value;
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

use crate::read_stdin_payload;

use super::{CODEX_CLI_UNKNOWN_EXIT_CODE, WorkerRequest, launch_spec};

pub(crate) struct WorkerSpawnInput {
    pub(crate) args: Vec<String>,
    pub(crate) working_dir: String,
    pub(crate) output_path: String,
    pub(crate) exit_code_path: String,
    pub(crate) codex_executable_path: Option<String>,
    pub(crate) node_executable_path: Option<String>,
}

pub(crate) fn run_worker() {
    let request = match read_worker_request() {
        Ok(request) => request,
        Err(err) => {
            eprintln!("read codex worker request failed: {err}");
            return;
        }
    };
    run_worker_request(request);
}

fn read_worker_request() -> Result<WorkerRequest, String> {
    let input = read_stdin_payload()?;
    serde_json::from_str(&input).map_err(|err| format!("invalid codex worker request: {err}"))
}

fn run_worker_request(request: WorkerRequest) {
    let exit_code = match execute_codex_worker(&request) {
        Ok(code) => code,
        Err(err) => {
            append_output_line(&request.output_path, &format!("worker error: {err}"));
            CODEX_CLI_UNKNOWN_EXIT_CODE
        }
    };
    if let Err(err) = write_exit_code(&request.exit_code_path, exit_code) {
        eprintln!("write codex worker exit code failed: {err}");
    }
}

fn execute_codex_worker(request: &WorkerRequest) -> Result<i32, String> {
    ensure_parent_exists(&request.output_path)?;
    ensure_parent_exists(&request.exit_code_path)?;
    let launch_spec = launch_spec::resolve_codex_launch_spec(
        request.codex_executable_path.as_deref(),
        request.node_executable_path.as_deref(),
    )?;
    let output_file = fs::File::create(&request.output_path)
        .map_err(|err| format!("create output file failed: {err}"))?;
    let error_file = output_file
        .try_clone()
        .map_err(|err| format!("clone output file failed: {err}"))?;
    let mut command = Command::new(&launch_spec.executable_path);
    if let Some(path) = launch_spec.path_override.as_ref() {
        command.env("PATH", path);
    }
    let status = command
        .args(&request.args)
        .current_dir(&request.working_dir)
        .stdout(Stdio::from(output_file))
        .stderr(Stdio::from(error_file))
        .status()
        .map_err(|err| format!("spawn codex failed: {err}"))?;
    Ok(status.code().unwrap_or(CODEX_CLI_UNKNOWN_EXIT_CODE))
}

pub(crate) fn spawn_worker_process(input: WorkerSpawnInput) -> Result<(), String> {
    let request = WorkerRequest {
        args: input.args,
        working_dir: input.working_dir,
        output_path: input.output_path,
        exit_code_path: input.exit_code_path,
        codex_executable_path: input.codex_executable_path,
        node_executable_path: input.node_executable_path,
    };
    let request_json = serde_json::to_string(&request)
        .map_err(|err| format!("encode worker request failed: {err}"))?;
    let current_exe =
        std::env::current_exe().map_err(|err| format!("resolve current exe failed: {err}"))?;
    let mut child = Command::new(current_exe)
        .arg("--codex-cli-worker")
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|err| format!("spawn worker failed: {err}"))?;
    let mut child_stdin = child
        .stdin
        .take()
        .ok_or_else(|| "worker stdin is unavailable".to_string())?;
    child_stdin
        .write_all(request_json.as_bytes())
        .map_err(|err| format!("write worker request failed: {err}"))?;
    Ok(())
}

pub(crate) fn read_output_tail(path: &str, max_chars: usize) -> Result<String, String> {
    if max_chars == 0 {
        return Ok(String::new());
    }
    let text = match fs::read_to_string(path) {
        Ok(text) => text,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(String::new()),
        Err(err) => return Err(format!("read output file failed: {err}")),
    };
    Ok(trim_to_last_chars(&text, max_chars))
}

pub(crate) fn read_final_agent_message(path: &str) -> Result<Option<String>, String> {
    let text = read_optional_output_file(path)?;
    if text.trim().is_empty() {
        return Ok(None);
    }
    extract_final_agent_message(&text)
}

fn read_optional_output_file(path: &str) -> Result<String, String> {
    match fs::read_to_string(path) {
        Ok(text) => Ok(text),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(String::new()),
        Err(err) => Err(format!("read output file failed: {err}")),
    }
}

fn extract_final_agent_message(text: &str) -> Result<Option<String>, String> {
    let mut final_message: Option<String> = None;
    for (index, line) in text.lines().enumerate() {
        if let Some(message) = parse_agent_message_line(line, index)? {
            final_message = Some(message);
        }
    }
    Ok(final_message)
}

fn parse_agent_message_line(line: &str, index: usize) -> Result<Option<String>, String> {
    let trimmed = line.trim();
    if !trimmed.starts_with('{') {
        return Ok(None);
    }
    let event: Value = serde_json::from_str(trimmed)
        .map_err(|err| format!("invalid codex JSON event at line {}: {err}", index + 1))?;
    Ok(extract_agent_message_text(&event))
}

fn extract_agent_message_text(event: &Value) -> Option<String> {
    if event.get("type")?.as_str()? != "item.completed" {
        return None;
    }
    let item = event.get("item")?;
    if item.get("type")?.as_str()? != "agent_message" {
        return None;
    }
    let text = item.get("text")?.as_str()?.trim();
    (!text.is_empty()).then(|| text.to_string())
}

pub(crate) fn read_exit_code(path: &str) -> Result<Option<i32>, String> {
    let text = match fs::read_to_string(path) {
        Ok(text) => text,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(err) => return Err(format!("read exit code file failed: {err}")),
    };
    let parsed = text
        .trim()
        .parse::<i32>()
        .map_err(|err| format!("invalid exit code file: {err}"))?;
    Ok(Some(parsed))
}

pub(crate) fn default_output_path() -> String {
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis();
    let mut path = PathBuf::from(std::env::temp_dir());
    path.push(format!("ghost-codex-cli-{stamp}.log"));
    path.to_string_lossy().to_string()
}

fn write_exit_code(path: &str, exit_code: i32) -> Result<(), String> {
    ensure_parent_exists(path)?;
    fs::write(path, exit_code.to_string())
        .map_err(|err| format!("write exit code file failed: {err}"))
}

fn ensure_parent_exists(path: &str) -> Result<(), String> {
    let parent = Path::new(path)
        .parent()
        .ok_or_else(|| format!("path has no parent: {path}"))?;
    fs::create_dir_all(parent).map_err(|err| format!("create parent directory failed: {err}"))
}

fn append_output_line(path: &str, message: &str) {
    let result = (|| -> Result<(), String> {
        ensure_parent_exists(path)?;
        let mut file = fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(path)
            .map_err(|err| format!("open output file failed: {err}"))?;
        writeln!(file, "{message}").map_err(|err| format!("append output file failed: {err}"))
    })();
    if let Err(err) = result {
        eprintln!("{err}");
    }
}

fn trim_to_last_chars(input: &str, max_chars: usize) -> String {
    if max_chars == 0 {
        return String::new();
    }
    let chars: Vec<char> = input.chars().collect();
    if chars.len() <= max_chars {
        return input.to_string();
    }
    chars[chars.len() - max_chars..].iter().collect()
}

#[cfg(test)]
mod tests {
    use super::read_final_agent_message;
    use serde_json::json;
    use std::fs;
    use std::path::PathBuf;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn read_final_agent_message_returns_last_non_empty_message() {
        let path = temp_output_path("final-message");
        let first = agent_message_event("first draft");
        let empty = agent_message_event("   ");
        let last = agent_message_event("done with checks");
        fs::write(&path, format!("warning\n{first}\n{empty}\n{last}\n"))
            .expect("output log should be written");

        let message = read_final_agent_message(path.to_str().expect("path must be utf8"))
            .expect("final message should parse");

        assert_eq!(message.as_deref(), Some("done with checks"));
    }

    #[test]
    fn read_final_agent_message_reports_invalid_json_event() {
        let path = temp_output_path("invalid-json");
        fs::write(&path, "{\"type\":\"item.completed\"\n").expect("output log should be written");

        let err = read_final_agent_message(path.to_str().expect("path must be utf8"))
            .expect_err("invalid JSON event should be explicit");

        assert!(err.contains("invalid codex JSON event at line 1"));
    }

    fn agent_message_event(text: &str) -> String {
        json!({
            "type": "item.completed",
            "item": {
                "type": "agent_message",
                "text": text
            }
        })
        .to_string()
    }

    fn temp_output_path(prefix: &str) -> PathBuf {
        let stamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos();
        let mut path = std::env::temp_dir();
        path.push(format!("ghost-native-codex-worker-{prefix}-{stamp}.log"));
        path
    }
}
