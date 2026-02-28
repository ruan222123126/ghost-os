#![allow(unsafe_op_in_unsafe_fn)]

mod sandbox;

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::fs;
use std::io::{self, BufReader, Read, Write};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use sandbox::{ExecutionResult, PythonSandbox, SandboxConfig};

// Request 对齐 core/shared/schema.json 的请求结构。
#[derive(Deserialize)]
struct Request {
    action: String,
    #[serde(rename = "params", default)]
    params: Value,
    #[serde(rename = "trace_id", default)]
    trace_id: String,
}

// Response 是 native 层统一返回格式。
#[derive(Serialize)]
struct Response {
    status: String,
    payload: Value,
    error: String,
}

impl Response {
    // success 构造成功响应。
    fn success(payload: Value) -> Self {
        Self {
            status: "success".to_string(),
            payload,
            error: String::new(),
        }
    }

    // error 构造失败响应。
    fn error(message: String) -> Self {
        Self {
            status: "error".to_string(),
            payload: json!({}),
            error: message,
        }
    }
}

#[derive(Deserialize, Serialize)]
struct ScriptWorkerRequest {
    script: String,
    max_memory_mb: u64,
}

fn main() {
    if std::env::args().any(|arg| arg == "--sandbox-worker") {
        run_sandbox_worker();
        return;
    }

    // 读取整段 stdin，保持最小协议处理路径。
    let input = match read_stdin_payload() {
        Ok(input) => input,
        Err(err) => {
            emit(Response::error(err));
            return;
        }
    };

    let request: Request = match serde_json::from_str(&input) {
        Ok(request) => request,
        Err(err) => {
            emit(Response::error(format!("invalid json: {err}")));
            return;
        }
    };

    let response = match request.action.as_str() {
        "PING" => Response::success(json!({
            "message": "PONG",
            "trace_id": request.trace_id
        })),
        "LIST_FILES" => handle_list_files(&request.params),
        "BASH_EXEC" => handle_bash_exec(&request.params),
        "SCRIPT_EXEC" => handle_script_exec(&request.params),
        _ => Response::error(format!("unsupported action: {}", request.action)),
    };

    emit(response);
}

fn run_sandbox_worker() {
    let input = match read_stdin_payload() {
        Ok(input) => input,
        Err(err) => {
            emit_worker_result(ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(err),
            });
            return;
        }
    };

    let request: ScriptWorkerRequest = match serde_json::from_str(&input) {
        Ok(request) => request,
        Err(err) => {
            emit_worker_result(ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(format!("invalid sandbox worker request: {err}")),
            });
            return;
        }
    };

    if let Err(err) = apply_memory_limit(request.max_memory_mb) {
        emit_worker_result(ExecutionResult {
            output: String::new(),
            tool_calls_log: Vec::new(),
            error: Some(err),
        });
        return;
    }

    let mut config = SandboxConfig::default();
    config.max_memory_mb = request.max_memory_mb;
    let sandbox = PythonSandbox::new(config);
    let result = sandbox.execute_blocking(&request.script);
    emit_worker_result(result);
}

fn read_stdin_payload() -> Result<String, String> {
    let mut input = String::new();
    io::stdin()
        .read_to_string(&mut input)
        .map_err(|_| "failed to read stdin".to_string())?;

    let input = input.trim();
    if input.is_empty() {
        return Err("empty input".to_string());
    }

    Ok(input.to_string())
}

fn handle_list_files(params: &Value) -> Response {
    let path = params
        .get("path")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".");

    let read_dir = match fs::read_dir(path) {
        Ok(entries) => entries,
        Err(err) => {
            return Response::error(format!("read dir {path:?} failed: {err}"));
        }
    };

    let mut names = Vec::new();
    for entry_result in read_dir {
        let entry = match entry_result {
            Ok(entry) => entry,
            Err(err) => return Response::error(format!("read dir entry failed: {err}")),
        };

        let metadata = match entry.metadata() {
            Ok(metadata) => metadata,
            Err(err) => return Response::error(format!("read metadata failed: {err}")),
        };

        let mut name = entry.file_name().to_string_lossy().to_string();
        if metadata.is_dir() {
            name.push('/');
        }
        names.push(name);
    }

    names.sort();
    Response::success(json!({
        "path": path,
        "entries": names,
    }))
}

fn handle_bash_exec(params: &Value) -> Response {
    let command = params
        .get("command")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");

    if command.is_empty() {
        return Response::error("command is required".to_string());
    }

    // 实际 shell 执行策略尚未开放，这里保留 execution layer 受控扩展点。
    Response::success(json!({
        "output": format!("[stub] execution layer received command: {command}"),
    }))
}

fn handle_script_exec(params: &Value) -> Response {
    let script = params
        .get("script")
        .and_then(Value::as_str)
        .map(str::trim)
        .unwrap_or("");

    if script.is_empty() {
        return Response::error("script is required".to_string());
    }

    let mut timeout_ms = params
        .get("timeout_ms")
        .and_then(Value::as_u64)
        .unwrap_or(30_000);
    if timeout_ms == 0 {
        timeout_ms = 30_000;
    }
    if timeout_ms > 60_000 {
        timeout_ms = 60_000;
    }

    let mut max_memory_mb = params
        .get("max_memory_mb")
        .and_then(Value::as_u64)
        .unwrap_or(256);
    if max_memory_mb == 0 {
        max_memory_mb = 256;
    }
    if max_memory_mb > 512 {
        max_memory_mb = 512;
    }

    let result = match execute_script_in_subprocess(script, timeout_ms, max_memory_mb) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    if let Some(err) = result.error {
        return Response::error(err);
    }

    Response::success(json!({
        "output": result.output,
        "tool_calls_log": result.tool_calls_log,
    }))
}

fn execute_script_in_subprocess(
    script: &str,
    timeout_ms: u64,
    max_memory_mb: u64,
) -> Result<ExecutionResult, String> {
    let worker_request = ScriptWorkerRequest {
        script: script.to_string(),
        max_memory_mb,
    };

    let mut child = Command::new(
        std::env::current_exe()
            .map_err(|err| format!("resolve current executable failed: {err}"))?,
    )
    .arg("--sandbox-worker")
    .stdin(Stdio::piped())
    .stdout(Stdio::piped())
    .stderr(Stdio::piped())
    .spawn()
    .map_err(|err| format!("spawn sandbox worker failed: {err}"))?;

    let mut stdin = child
        .stdin
        .take()
        .ok_or_else(|| "sandbox worker stdin is not available".to_string())?;
    if let Err(err) = serde_json::to_writer(&mut stdin, &worker_request) {
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("encode sandbox worker request failed: {err}"));
    }
    if let Err(err) = stdin.write_all(b"\n") {
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("flush sandbox worker request failed: {err}"));
    }
    drop(stdin);

    let stdout = match child.stdout.take() {
        Some(stdout) => stdout,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err("sandbox worker stdout is not available".to_string());
        }
    };
    let stderr = match child.stderr.take() {
        Some(stderr) => stderr,
        None => {
            let _ = child.kill();
            let _ = child.wait();
            return Err("sandbox worker stderr is not available".to_string());
        }
    };

    let stdout_handle = spawn_pipe_reader(stdout);
    let stderr_handle = spawn_pipe_reader(stderr);

    let status = match wait_child_with_timeout(&mut child, Duration::from_millis(timeout_ms)) {
        Ok(Some(status)) => status,
        Ok(None) => {
            let _ = stdout_handle.join();
            let _ = stderr_handle.join();
            return Err(format!("script execution timeout after {}ms", timeout_ms));
        }
        Err(err) => {
            let _ = stdout_handle.join();
            let _ = stderr_handle.join();
            return Err(err);
        }
    };

    let stdout_bytes = stdout_handle.join().unwrap_or_default();
    let stderr_bytes = stderr_handle.join().unwrap_or_default();

    if !status.success() {
        let stderr_text = String::from_utf8_lossy(&stderr_bytes);
        let stderr_line = stderr_text.lines().next().unwrap_or("").trim();
        if stderr_line.is_empty() {
            return Err(format!("sandbox worker exited with status {}", status));
        }
        return Err(format!(
            "sandbox worker exited with status {}: {}",
            status, stderr_line
        ));
    }

    serde_json::from_slice::<ExecutionResult>(&stdout_bytes).map_err(|err| {
        let stdout_text = String::from_utf8_lossy(&stdout_bytes);
        let stdout_line = stdout_text.lines().next().unwrap_or("").trim();
        if stdout_line.is_empty() {
            format!("decode sandbox worker response failed: {err}")
        } else {
            format!(
                "decode sandbox worker response failed: {} (stdout: {})",
                err, stdout_line
            )
        }
    })
}

fn spawn_pipe_reader<R>(reader: R) -> thread::JoinHandle<Vec<u8>>
where
    R: Read + Send + 'static,
{
    thread::spawn(move || {
        let mut buffer = Vec::new();
        let mut reader = BufReader::new(reader);
        let _ = reader.read_to_end(&mut buffer);
        buffer
    })
}

fn wait_child_with_timeout(
    child: &mut Child,
    timeout: Duration,
) -> Result<Option<ExitStatus>, String> {
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait() {
            Ok(Some(status)) => return Ok(Some(status)),
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Ok(None);
                }
                thread::sleep(Duration::from_millis(10));
            }
            Err(err) => return Err(format!("wait sandbox worker failed: {err}")),
        }
    }
}

#[cfg(unix)]
fn apply_memory_limit(max_memory_mb: u64) -> Result<(), String> {
    let memory_bytes = max_memory_mb
        .checked_mul(1024)
        .and_then(|value| value.checked_mul(1024))
        .ok_or_else(|| "max_memory_mb is too large".to_string())?;

    if (memory_bytes as u128) > (libc::rlim_t::MAX as u128) {
        return Err("max_memory_mb exceeds platform limit".to_string());
    }

    let rlim = memory_bytes as libc::rlim_t;
    let limit = libc::rlimit {
        rlim_cur: rlim,
        rlim_max: rlim,
    };

    // 使用进程级地址空间限制，确保沙盒超限时被系统拒绝继续分配内存。
    let rc = unsafe { libc::setrlimit(libc::RLIMIT_AS, &limit) };
    if rc != 0 {
        return Err(format!(
            "failed to apply memory limit: {}",
            io::Error::last_os_error()
        ));
    }

    Ok(())
}

#[cfg(not(unix))]
fn apply_memory_limit(_max_memory_mb: u64) -> Result<(), String> {
    Ok(())
}

// emit 负责写出单行 JSON 响应，失败时静默返回。
fn emit(response: Response) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&response) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}

fn emit_worker_result(result: ExecutionResult) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&result) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}
