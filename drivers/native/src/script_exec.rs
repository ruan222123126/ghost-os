use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::io::{self, BufReader, Read, Write};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use crate::sandbox::{ExecutionResult, PythonSandbox, SandboxConfig};
use crate::{Response, read_stdin_payload};

#[derive(Deserialize, Serialize)]
struct ScriptWorkerRequest {
    script: String,
    max_memory_mb: u64,
    sandbox_config: SandboxConfig,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct ScriptExecutionBudget {
    timeout_ms: u64,
    max_memory_mb: u64,
}

struct WorkerProcess {
    child: Child,
    stdout_handle: thread::JoinHandle<Vec<u8>>,
    stderr_handle: thread::JoinHandle<Vec<u8>>,
}

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "SCRIPT_EXEC" => Some(handle_script_exec(params)),
        _ => None,
    }
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

    let sandbox = PythonSandbox::new(request.sandbox_config);
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

    let sandbox_config = SandboxConfig::default();
    let budget = match resolve_script_budget(params, &sandbox_config) {
        Ok(budget) => budget,
        Err(err) => return Response::error(err),
    };

    let result = match execute_script_in_subprocess(script, budget, sandbox_config) {
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
    budget: ScriptExecutionBudget,
    sandbox_config: SandboxConfig,
) -> Result<ExecutionResult, String> {
    let worker_request = ScriptWorkerRequest {
        script: script.to_string(),
        max_memory_mb: budget.max_memory_mb,
        sandbox_config,
    };

    let WorkerProcess {
        mut child,
        stdout_handle,
        stderr_handle,
    } = spawn_worker_process(&worker_request)?;

    let status = match wait_child_with_timeout(&mut child, Duration::from_millis(budget.timeout_ms))
    {
        Ok(Some(status)) => status,
        Ok(None) => {
            let _ = stdout_handle.join();
            let _ = stderr_handle.join();
            return Err(format!(
                "script execution timeout after {}ms",
                budget.timeout_ms
            ));
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

fn resolve_script_budget(
    params: &Value,
    config: &SandboxConfig,
) -> Result<ScriptExecutionBudget, String> {
    let timeout_ms = clamp_budget(
        optional_u64(params, "timeout_ms")?,
        config.default_script_timeout_ms,
        config.max_script_timeout_ms,
    );
    let max_memory_mb = clamp_budget(
        optional_u64(params, "max_memory_mb")?,
        config.default_script_memory_mb,
        config.max_script_memory_mb,
    );

    Ok(ScriptExecutionBudget {
        timeout_ms,
        max_memory_mb,
    })
}

fn optional_u64(params: &Value, field: &str) -> Result<Option<u64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    raw.as_u64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a non-negative integer"))
}

fn clamp_budget(requested: Option<u64>, default_value: u64, max_value: u64) -> u64 {
    let mut value = requested.unwrap_or(default_value);
    if value == 0 {
        value = default_value;
    }
    value.min(max_value)
}

fn spawn_worker_process(worker_request: &ScriptWorkerRequest) -> Result<WorkerProcess, String> {
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
    if let Err(err) = serde_json::to_writer(&mut stdin, worker_request) {
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

    Ok(WorkerProcess {
        child,
        stdout_handle: spawn_pipe_reader(stdout),
        stderr_handle: spawn_pipe_reader(stderr),
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

#[cfg(test)]
mod tests {
    use super::{clamp_budget, dispatch_action, optional_u64, resolve_script_budget};
    use crate::sandbox::SandboxConfig;
    use serde_json::json;

    #[test]
    fn resolve_script_budget_uses_defaults_and_caps() {
        let config = SandboxConfig::default();
        let budget = resolve_script_budget(
            &json!({
                "timeout_ms": 120_000,
                "max_memory_mb": 2_048
            }),
            &config,
        )
        .expect("budget should resolve");

        assert_eq!(budget.timeout_ms, config.max_script_timeout_ms);
        assert_eq!(budget.max_memory_mb, config.max_script_memory_mb);
    }

    #[test]
    fn resolve_script_budget_treats_zero_as_default() {
        let config = SandboxConfig::default();
        let budget = resolve_script_budget(
            &json!({
                "timeout_ms": 0,
                "max_memory_mb": 0
            }),
            &config,
        )
        .expect("budget should resolve");

        assert_eq!(budget.timeout_ms, config.default_script_timeout_ms);
        assert_eq!(budget.max_memory_mb, config.default_script_memory_mb);
    }

    #[test]
    fn optional_u64_rejects_invalid_types() {
        let err = optional_u64(&json!({"timeout_ms": "slow"}), "timeout_ms")
            .expect_err("string value should fail");
        assert_eq!(err, "timeout_ms must be a non-negative integer");
    }

    #[test]
    fn clamp_budget_returns_default_for_none() {
        assert_eq!(clamp_budget(None, 10, 20), 10);
    }

    #[test]
    fn dispatch_action_returns_none_for_unknown_script_action() {
        assert!(dispatch_action("BASH_EXEC", &json!({})).is_none());
    }
}
