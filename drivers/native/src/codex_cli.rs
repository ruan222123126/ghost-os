use crate::Response;
use crate::sandbox::SandboxConfig;
use crate::sandbox::path_policy::{resolve_read_path, resolve_write_path};
use regex::Regex;
use serde_json::{Value, json};
use std::collections::HashMap;
use std::io::{BufRead, BufReader, Read};
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex, OnceLock};
use std::thread;
use std::time::{Duration, Instant};

const DEFAULT_WAIT_MS_BEFORE_ASYNC: u64 = 3000;
const DEFAULT_WAIT_DURATION_SECONDS: u64 = 300;
const DEFAULT_OUTPUT_CHAR_COUNT: usize = 200;
const MAX_OUTPUT_BUFFER_CHARS: usize = 4000;

pub(crate) struct CodexCommandManager {
    next_id: u64,
    commands: HashMap<String, CodexCommand>,
}

impl CodexCommandManager {
    pub(crate) fn new() -> Self {
        Self {
            next_id: 0,
            commands: HashMap::new(),
        }
    }

    fn next_command_id(&mut self) -> (u64, String) {
        self.next_id += 1;
        let id = format!("codex-cli-{}", self.next_id);
        (self.next_id, id)
    }

    fn insert(&mut self, cmd: CodexCommand) {
        self.commands.insert(cmd.id.clone(), cmd);
    }

    fn get_mut(&mut self, id: &str) -> Option<&mut CodexCommand> {
        self.commands.get_mut(id)
    }

    fn find_by_session_id(&mut self, session_id: &str) -> Option<&mut CodexCommand> {
        let mut best_id: Option<String> = None;
        let mut best_seq = 0;
        for (id, cmd) in self.commands.iter() {
            if cmd.session_id_value().as_deref() == Some(session_id) {
                if cmd.seq > best_seq {
                    best_seq = cmd.seq;
                    best_id = Some(id.clone());
                }
            }
        }
        best_id.and_then(|id| self.commands.get_mut(&id))
    }
}

struct CodexCommand {
    seq: u64,
    id: String,
    child: Child,
    output: Arc<Mutex<OutputBuffer>>,
    session_id: Arc<Mutex<Option<String>>>,
    output_path: Option<String>,
    exit_code: Option<i32>,
}

impl CodexCommand {
    fn session_id_value(&self) -> Option<String> {
        self.session_id.lock().ok().and_then(|guard| guard.clone())
    }

    fn output_tail(&self, max_chars: usize) -> String {
        let buffer = match self.output.lock() {
            Ok(buffer) => buffer.snapshot(),
            Err(_) => String::new(),
        };
        trim_to_last_chars(&buffer, max_chars)
    }

    fn update_exit_code(&mut self) -> Result<Option<i32>, String> {
        if let Some(code) = self.exit_code {
            return Ok(Some(code));
        }
        match self.child.try_wait() {
            Ok(Some(status)) => {
                let code = status.code().unwrap_or(-1);
                self.exit_code = Some(code);
                Ok(Some(code))
            }
            Ok(None) => Ok(None),
            Err(err) => Err(format!("check process status failed: {err}")),
        }
    }
}

#[derive(Default)]
struct OutputBuffer {
    data: String,
}

impl OutputBuffer {
    fn push(&mut self, text: &str) {
        if text.is_empty() {
            return;
        }
        self.data.push_str(text);
        if self.data.chars().count() > MAX_OUTPUT_BUFFER_CHARS {
            self.data = trim_to_last_chars(&self.data, MAX_OUTPUT_BUFFER_CHARS);
        }
    }

    fn snapshot(&self) -> String {
        self.data.clone()
    }
}

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
        "CODEX_CLI_START" => Some(handle_start(params, state)),
        "CODEX_CLI_STATUS" => Some(handle_status(params, state)),
        _ => None,
    }
}

fn handle_start(params: &Value, state: &mut CodexCommandManager) -> Response {
    let op = match parse_required_string(params, "op") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let prompt = match parse_required_string(params, "prompt") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let session_id = match parse_optional_string(params, "session_id") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let cwd = match parse_optional_string(params, "cwd") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let output_path = match parse_optional_string(params, "output_path") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let model = match parse_optional_string(params, "model") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let full_auto = match parse_optional_bool(params, "full_auto") {
        Ok(value) => value.unwrap_or(true),
        Err(err) => return Response::error(err),
    };
    let skip_git_repo_check = match parse_optional_bool(params, "skip_git_repo_check") {
        Ok(value) => value.unwrap_or(true),
        Err(err) => return Response::error(err),
    };
    let json_flag = match parse_optional_bool(params, "json") {
        Ok(value) => value.unwrap_or(true),
        Err(err) => return Response::error(err),
    };
    let wait_ms_before_async = match parse_optional_u64(params, "wait_ms_before_async") {
        Ok(value) => value.unwrap_or(DEFAULT_WAIT_MS_BEFORE_ASYNC),
        Err(err) => return Response::error(err),
    };
    let output_char_count = match parse_optional_u64(params, "output_character_count") {
        Ok(value) => value.unwrap_or(DEFAULT_OUTPUT_CHAR_COUNT as u64),
        Err(err) => return Response::error(err),
    };

    let op = op.to_lowercase();
    if !matches!(op.as_str(), "start" | "resume" | "fork") {
        return Response::error("op must be one of: start, resume, fork".to_string());
    }
    if op != "start" && session_id.as_deref().unwrap_or("").is_empty() {
        return Response::error("session_id is required for resume/fork".to_string());
    }

    let config = SandboxConfig::default();
    let resolved_cwd = match cwd.as_deref() {
        Some(raw) if !raw.trim().is_empty() => match resolve_read_path(raw, &config) {
            Ok(path) => Some(path),
            Err(err) => return Response::error(err),
        },
        _ => None,
    };
    let resolved_output = match output_path.as_deref() {
        Some(raw) if !raw.trim().is_empty() => match resolve_write_path(raw, &config) {
            Ok(path) => Some(path),
            Err(err) => return Response::error(err),
        },
        _ => None,
    };

    let mut cmd = Command::new("codex");
    cmd.stdout(Stdio::piped()).stderr(Stdio::piped());
    if let Some(dir) = resolved_cwd.as_ref() {
        cmd.current_dir(dir);
    }

    match op.as_str() {
        "start" => {
            cmd.arg("exec").arg(prompt.trim());
        }
        "resume" => {
            cmd.arg("exec")
                .arg("resume")
                .arg("--session-id")
                .arg(session_id.clone().unwrap_or_default())
                .arg(prompt.trim());
        }
        "fork" => {
            cmd.arg("fork")
                .arg("--session-id")
                .arg(session_id.clone().unwrap_or_default())
                .arg(prompt.trim());
        }
        _ => {}
    }

    if full_auto {
        cmd.arg("--full-auto");
    }
    if skip_git_repo_check {
        cmd.arg("--skip-git-repo-check");
    }
    if json_flag {
        cmd.arg("--json");
    }
    if let Some(model) = model {
        if !model.trim().is_empty() {
            cmd.arg("-m").arg(model.trim());
        }
    }
    if let Some(dir) = resolved_cwd.as_ref() {
        cmd.arg("-C").arg(dir);
    }
    if let Some(path) = resolved_output.as_ref() {
        cmd.arg("-o").arg(path);
    }

    let mut child = match cmd.spawn() {
        Ok(child) => child,
        Err(err) => {
            return Response::success(json!({
                "status": "error",
                "message": format!("spawn codex failed: {err}"),
            }));
        }
    };

    let stdout = child.stdout.take();
    let stderr = child.stderr.take();

    let output = Arc::new(Mutex::new(OutputBuffer::default()));
    let session_id_state = Arc::new(Mutex::new(None));
    spawn_output_reader(stdout, output.clone(), session_id_state.clone(), "STDOUT: ");
    spawn_output_reader(stderr, output.clone(), session_id_state.clone(), "STDERR: ");

    let (seq, command_id) = state.next_command_id();
    let mut command = CodexCommand {
        seq,
        id: command_id.clone(),
        child,
        output: output.clone(),
        session_id: session_id_state.clone(),
        output_path: resolved_output
            .as_ref()
            .map(|path: &std::path::PathBuf| path.to_string_lossy().to_string()),
        exit_code: None,
    };

    if wait_ms_before_async > 0 {
        thread::sleep(Duration::from_millis(wait_ms_before_async));
    }

    let mut status = "running".to_string();
    let mut exit_code: Option<i32> = None;
    match command.update_exit_code() {
        Ok(Some(code)) => {
            status = "done".to_string();
            exit_code = Some(code);
        }
        Ok(None) => {}
        Err(err) => {
            status = "error".to_string();
            return Response::success(json!({
                "status": status,
                "command_id": command_id,
                "session_id": command.session_id_value().unwrap_or_default(),
                "output_tail": command.output_tail(output_char_count as usize),
                "output_path": command.output_path.clone().unwrap_or_default(),
                "message": err,
            }));
        }
    }

    let output_tail = command.output_tail(output_char_count as usize);
    let session_value = command.session_id_value().unwrap_or_default();
    let output_path_value = command.output_path.clone().unwrap_or_default();
    state.insert(command);

    Response::success(json!({
        "status": status,
        "command_id": command_id,
        "session_id": session_value,
        "exit_code": exit_code,
        "output_tail": output_tail,
        "output_path": output_path_value,
    }))
}

fn handle_status(params: &Value, state: &mut CodexCommandManager) -> Response {
    let command_id = match parse_optional_string(params, "command_id") {
        Ok(value) => value.unwrap_or_default(),
        Err(err) => return Response::error(err),
    };
    let session_id = match parse_optional_string(params, "session_id") {
        Ok(value) => value.unwrap_or_default(),
        Err(err) => return Response::error(err),
    };
    if command_id.trim().is_empty() && session_id.trim().is_empty() {
        return Response::error("command_id or session_id is required".to_string());
    }

    let wait_duration_seconds = match parse_optional_u64(params, "wait_duration_seconds") {
        Ok(value) => value.unwrap_or(DEFAULT_WAIT_DURATION_SECONDS),
        Err(err) => return Response::error(err),
    };
    let output_char_count = match parse_optional_u64(params, "output_character_count") {
        Ok(value) => value.unwrap_or(DEFAULT_OUTPUT_CHAR_COUNT as u64),
        Err(err) => return Response::error(err),
    } as usize;

    let command = if !command_id.trim().is_empty() {
        if let Some(command) = state.get_mut(command_id.trim()) {
            Some(command)
        } else {
            state.find_by_session_id(session_id.trim())
        }
    } else if !session_id.trim().is_empty() {
        state.find_by_session_id(session_id.trim())
    } else {
        None
    };
    let Some(command) = command else {
        return Response::success(json!({
            "status": "error",
            "message": "command not found",
        }));
    };

    let deadline = Instant::now() + Duration::from_secs(wait_duration_seconds);
    let mut status = "running".to_string();
    let mut exit_code: Option<i32> = None;

    loop {
        match command.update_exit_code() {
            Ok(Some(code)) => {
                status = "done".to_string();
                exit_code = Some(code);
                break;
            }
            Ok(None) => {}
            Err(err) => {
                status = "error".to_string();
                return Response::success(json!({
                    "status": status,
                    "command_id": command.id,
                    "session_id": command.session_id_value().unwrap_or_default(),
                    "output_tail": command.output_tail(output_char_count),
                    "output_path": command.output_path.clone().unwrap_or_default(),
                    "message": err,
                }));
            }
        }
        if Instant::now() >= deadline {
            break;
        }
        thread::sleep(Duration::from_millis(500));
    }

    Response::success(json!({
        "status": status,
        "command_id": command.id,
        "session_id": command.session_id_value().unwrap_or_default(),
        "exit_code": exit_code,
        "output_tail": command.output_tail(output_char_count),
        "output_path": command.output_path.clone().unwrap_or_default(),
    }))
}

fn spawn_output_reader<R: Read + Send + 'static>(
    reader: Option<R>,
    output: Arc<Mutex<OutputBuffer>>,
    session_id: Arc<Mutex<Option<String>>>,
    prefix: &str,
) {
    let Some(reader) = reader else { return };
    let prefix = prefix.to_string();
    thread::spawn(move || {
        let mut reader = BufReader::new(reader);
        let mut line = String::new();
        loop {
            line.clear();
            match reader.read_line(&mut line) {
                Ok(0) => break,
                Ok(_) => {
                    let trimmed = line.trim_end_matches(&['\r', '\n'][..]);
                    if let Some(parsed) = parse_session_id(trimmed) {
                        if let Ok(mut guard) = session_id.lock() {
                            if guard.is_none() {
                                *guard = Some(parsed);
                            }
                        }
                    }
                    let mut buffer = match output.lock() {
                        Ok(buffer) => buffer,
                        Err(_) => return,
                    };
                    buffer.push(&format!("{prefix}{line}"));
                }
                Err(_) => break,
            }
        }
    });
}

fn parse_session_id(line: &str) -> Option<String> {
    let trimmed = line.trim();
    if trimmed.is_empty() {
        return None;
    }
    if trimmed.starts_with('{') && trimmed.ends_with('}') {
        if let Ok(value) = serde_json::from_str::<Value>(trimmed) {
            if let Some(id) = value.get("session_id").and_then(Value::as_str) {
                if !id.trim().is_empty() {
                    return Some(id.trim().to_string());
                }
            }
            if let Some(id) = value.get("sessionId").and_then(Value::as_str) {
                if !id.trim().is_empty() {
                    return Some(id.trim().to_string());
                }
            }
        }
    }

    for regex in session_id_patterns() {
        if let Some(caps) = regex.captures(trimmed) {
            if let Some(id) = caps.get(1) {
                let value = id.as_str().trim();
                if !value.is_empty() {
                    return Some(value.to_string());
                }
            }
        }
    }
    None
}

fn session_id_patterns() -> &'static Vec<Regex> {
    static PATTERNS: OnceLock<Vec<Regex>> = OnceLock::new();
    PATTERNS.get_or_init(|| {
        vec![
            Regex::new(r"(?i)session[_\s-]*id[:=]\s*([A-Za-z0-9_-]{6,})").unwrap(),
            Regex::new(r"(?i)session\s+id[:=]\s*([A-Za-z0-9_-]{6,})").unwrap(),
            Regex::new(r"(?i)session\s+([0-9a-fA-F-]{8,})").unwrap(),
        ]
    })
}

fn parse_required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn parse_optional_string(params: &Value, field: &str) -> Result<Option<String>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_str()
        .ok_or_else(|| format!("{field} must be a string"))?
        .trim();
    if value.is_empty() {
        return Ok(None);
    }
    Ok(Some(value.to_string()))
}

fn parse_optional_bool(params: &Value, field: &str) -> Result<Option<bool>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_bool()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a boolean"))
}

fn parse_optional_u64(params: &Value, field: &str) -> Result<Option<u64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_u64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a non-negative integer"))
}

fn trim_to_last_chars(input: &str, max_chars: usize) -> String {
    if max_chars == 0 {
        return String::new();
    }
    let total = input.chars().count();
    if total <= max_chars {
        return input.to_string();
    }
    let skip = total - max_chars;
    input.chars().skip(skip).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_start_requires_prompt() {
        let mut manager = CodexCommandManager::new();
        let response = handle_start(&json!({"op": "start"}), &mut manager);
        assert_eq!(response.status, "error");
    }

    #[test]
    fn test_start_rejects_disallowed_cwd() {
        let mut manager = CodexCommandManager::new();
        let response = handle_start(
            &json!({"op": "start", "prompt": "hi", "cwd": "/ghost-os/does-not-exist"}),
            &mut manager,
        );
        assert_eq!(response.status, "error");
    }

    #[test]
    fn test_status_unknown_command_id_returns_error_status() {
        let mut manager = CodexCommandManager::new();
        let response = handle_status(&json!({"command_id": "missing"}), &mut manager);
        assert_eq!(response.status, "success");
        let status = response
            .payload
            .get("status")
            .and_then(Value::as_str)
            .unwrap_or("");
        assert_eq!(status, "error");
    }
}
