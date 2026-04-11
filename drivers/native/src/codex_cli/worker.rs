use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

use crate::read_stdin_payload;

use super::{CODEX_CLI_EXECUTABLE, CODEX_CLI_UNKNOWN_EXIT_CODE, WorkerRequest};

pub(crate) struct WorkerSpawnInput {
    pub(crate) args: Vec<String>,
    pub(crate) working_dir: String,
    pub(crate) output_path: String,
    pub(crate) exit_code_path: String,
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
    let output_file = fs::File::create(&request.output_path)
        .map_err(|err| format!("create output file failed: {err}"))?;
    let error_file = output_file
        .try_clone()
        .map_err(|err| format!("clone output file failed: {err}"))?;
    let status = Command::new(CODEX_CLI_EXECUTABLE)
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
