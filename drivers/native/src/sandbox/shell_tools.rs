#[cfg(feature = "python-sandbox")]
use pyo3::prelude::*;
#[cfg(feature = "python-sandbox")]
use serde_json::json;
use std::io::{self, Read};
use std::process::{Child, Command, Output, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use super::SandboxConfig;
#[cfg(feature = "python-sandbox")]
use super::tool_runtime::ToolRuntime;

pub(crate) struct BashExecOutput {
    pub(crate) stdout: String,
    pub(crate) stderr: String,
    pub(crate) status: String,
    pub(crate) success: bool,
}

#[cfg(feature = "python-sandbox")]
pub(crate) fn bash_exec_py(
    runtime: &ToolRuntime,
    py: Python<'_>,
    command: String,
    max_output_chars: Option<usize>,
) -> PyResult<String> {
    let command = command.trim().to_string();
    let args = bash_exec_args(&command, max_output_chars);
    if command.is_empty() {
        return Err(runtime.log_error("bash_exec", args, "command is required"));
    }

    match py
        .allow_threads(|| bash_exec_impl(runtime.config(), &command, true, None, max_output_chars))
    {
        Ok(result) if result.success => {
            runtime.log_success("bash_exec", args, result.stdout.clone());
            Ok(result.stdout)
        }
        Ok(result) => {
            let err = format_bash_exec_failure(&result);
            Err(runtime.log_failure("bash_exec", args, result.stdout, err))
        }
        Err(err) => Err(runtime.log_error("bash_exec", args, err)),
    }
}

#[cfg(feature = "python-sandbox")]
pub(crate) fn bash_exec_result_py(
    runtime: &ToolRuntime,
    py: Python<'_>,
    command: String,
    timeout_ms: Option<u64>,
    max_output_chars: Option<usize>,
) -> PyResult<String> {
    let command = command.trim().to_string();
    let args = bash_exec_result_args(&command, timeout_ms, max_output_chars);
    if command.is_empty() {
        return Err(runtime.log_error("bash_exec", args, "command is required"));
    }

    match py.allow_threads(|| {
        bash_exec_impl(
            runtime.config(),
            &command,
            true,
            timeout_ms,
            max_output_chars,
        )
    }) {
        Ok(result) => {
            let error = (!result.success).then(|| format_bash_exec_failure(&result));
            runtime.record_tool_call("bash_exec", args, result.stdout.clone(), error);
            Ok(serialize_bash_exec_result(&result))
        }
        Err(err) => Err(runtime.log_error("bash_exec", args, err)),
    }
}

pub(crate) fn bash_exec_impl(
    config: &SandboxConfig,
    command: &str,
    login: bool,
    timeout_ms: Option<u64>,
    max_output_chars: Option<usize>,
) -> Result<BashExecOutput, String> {
    let command = command.trim();
    if command.is_empty() {
        return Err("command is required".to_string());
    }
    let output_limit = resolve_shell_output_chars(config, max_output_chars)?;

    let output = run_shell_command(command, login, resolve_shell_timeout(config, timeout_ms))
        .map_err(|err| {
            if err.kind() == io::ErrorKind::TimedOut {
                err.to_string()
            } else {
                format!("failed to execute command: {err}")
            }
        })?;

    Ok(BashExecOutput {
        stdout: truncate_shell_output(&String::from_utf8_lossy(&output.stdout), output_limit),
        stderr: truncate_shell_output(&String::from_utf8_lossy(&output.stderr), output_limit),
        status: output.status.to_string(),
        success: output.status.success(),
    })
}

#[cfg(feature = "python-sandbox")]
fn bash_exec_args(command: &str, max_output_chars: Option<usize>) -> serde_json::Value {
    match max_output_chars {
        Some(limit) => json!({ "command": command, "max_output_chars": limit }),
        None => json!({ "command": command }),
    }
}

#[cfg(feature = "python-sandbox")]
fn bash_exec_result_args(
    command: &str,
    timeout_ms: Option<u64>,
    max_output_chars: Option<usize>,
) -> serde_json::Value {
    match (timeout_ms, max_output_chars) {
        (Some(timeout), Some(limit)) => {
            json!({ "command": command, "timeout_ms": timeout, "max_output_chars": limit })
        }
        (Some(timeout), None) => json!({ "command": command, "timeout_ms": timeout }),
        (None, Some(limit)) => json!({ "command": command, "max_output_chars": limit }),
        (None, None) => json!({ "command": command }),
    }
}

#[cfg(feature = "python-sandbox")]
fn serialize_bash_exec_result(result: &BashExecOutput) -> String {
    json!({
        "stdout": result.stdout,
        "stderr": result.stderr,
        "status": result.status,
        "success": result.success,
    })
    .to_string()
}

pub(crate) fn format_bash_exec_failure(result: &BashExecOutput) -> String {
    if result.stderr.trim().is_empty() {
        format!("command failed: {}", result.status)
    } else {
        format!("command failed: {}", result.stderr.trim())
    }
}

pub(crate) fn resolve_shell_timeout(
    config: &SandboxConfig,
    requested_timeout_ms: Option<u64>,
) -> u64 {
    let mut timeout_ms = requested_timeout_ms.unwrap_or(config.default_shell_timeout_ms);
    if timeout_ms == 0 {
        timeout_ms = config.default_shell_timeout_ms;
    }
    timeout_ms.min(config.max_shell_timeout_ms)
}

pub(crate) fn resolve_shell_output_chars(
    config: &SandboxConfig,
    requested_max_output_chars: Option<usize>,
) -> Result<usize, String> {
    match requested_max_output_chars {
        Some(0) => Err("max_output_chars must be greater than 0".to_string()),
        Some(limit) => Ok(limit),
        None => Ok(config.max_shell_output_chars),
    }
}

fn run_shell_command(command: &str, login: bool, timeout_ms: u64) -> io::Result<Output> {
    let mut process = if cfg!(target_os = "windows") {
        let mut process = Command::new("cmd");
        process.arg("/C").arg(command);
        process
    } else {
        let mut process = Command::new("bash");
        if login {
            process.arg("-lc");
        } else {
            process.arg("-c");
        }
        process.arg(command);
        process
    };

    process.stdout(Stdio::piped()).stderr(Stdio::piped());
    let child = process.spawn()?;
    wait_shell_command_with_timeout(child, Duration::from_millis(timeout_ms))
}

fn wait_shell_command_with_timeout(mut child: Child, timeout: Duration) -> io::Result<Output> {
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| io::Error::other("command stdout is not available"))?;
    let stderr = child
        .stderr
        .take()
        .ok_or_else(|| io::Error::other("command stderr is not available"))?;

    let stdout_handle = spawn_pipe_reader(stdout);
    let stderr_handle = spawn_pipe_reader(stderr);
    let deadline = Instant::now() + timeout;

    loop {
        match child.try_wait() {
            Ok(Some(status)) => {
                let stdout = stdout_handle.join().unwrap_or_default();
                let stderr = stderr_handle.join().unwrap_or_default();
                return Ok(Output {
                    status,
                    stdout,
                    stderr,
                });
            }
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let _ = stdout_handle.join();
                    let _ = stderr_handle.join();
                    return Err(io::Error::new(
                        io::ErrorKind::TimedOut,
                        format!("command timed out after {}ms", timeout.as_millis()),
                    ));
                }
                thread::sleep(Duration::from_millis(10));
            }
            Err(err) => {
                let _ = stdout_handle.join();
                let _ = stderr_handle.join();
                return Err(err);
            }
        }
    }
}

fn spawn_pipe_reader<R>(reader: R) -> thread::JoinHandle<Vec<u8>>
where
    R: Read + Send + 'static,
{
    thread::spawn(move || {
        let mut buffer = Vec::new();
        let mut reader = io::BufReader::new(reader);
        let _ = reader.read_to_end(&mut buffer);
        buffer
    })
}

pub(crate) fn truncate_shell_output(text: &str, max_output_chars: usize) -> String {
    let char_count = text.chars().count();
    if char_count <= max_output_chars {
        return text.to_string();
    }

    let truncated: String = text.chars().take(max_output_chars).collect();
    format!(
        "{}\n\n[... output truncated, {} more chars. Use grep/head or write to file for full output]",
        truncated,
        char_count - max_output_chars
    )
}
