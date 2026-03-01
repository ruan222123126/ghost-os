use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::io::{self, BufReader, Read, Write};
use std::process::{Child, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use crate::sandbox::{ExecutionResult, PythonSandbox, SandboxConfig};
use crate::{Response, read_stdin_payload};

#[derive(Deserialize, Serialize)]
struct ScriptWorkerRequest {
    script: String,
    max_memory_mb: u64,
}

pub(crate) fn run_sandbox_worker() {
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

    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking(&request.script);
    emit_worker_result(result);
}

pub(crate) fn handle_script_exec(params: &Value) -> Response {
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

    let mut child = std::process::Command::new(
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

fn emit_worker_result(result: ExecutionResult) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&result) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}
