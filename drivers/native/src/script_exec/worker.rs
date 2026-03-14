use std::io::{self, BufReader, Read, Write};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant};

use crate::sandbox::{ExecutionResult, SandboxConfig};

use super::result::{decode_worker_result, worker_exit_error};
use super::types::{ScriptExecutionBudget, ScriptWorkerRequest, WorkerProcess};

pub(super) fn execute_script_in_subprocess(
    script: &str,
    budget: ScriptExecutionBudget,
    sandbox_config: SandboxConfig,
) -> Result<ExecutionResult, String> {
    let worker_request = ScriptWorkerRequest {
        script: script.to_string(),
        max_memory_mb: budget.max_memory_mb,
        sandbox_config,
    };
    let worker = spawn_worker_process(&worker_request)?;
    finish_worker_process(worker, budget.timeout_ms)
}

fn finish_worker_process(
    worker: WorkerProcess,
    timeout_ms: u64,
) -> Result<ExecutionResult, String> {
    let WorkerProcess {
        mut child,
        stdout_handle,
        stderr_handle,
    } = worker;

    let status = wait_child_with_timeout(&mut child, Duration::from_millis(timeout_ms));
    let (stdout_bytes, stderr_bytes) = join_worker_output(stdout_handle, stderr_handle);
    let status = status?;
    decode_completed_worker(status, timeout_ms, &stdout_bytes, &stderr_bytes)
}

fn decode_completed_worker(
    status: Option<ExitStatus>,
    timeout_ms: u64,
    stdout_bytes: &[u8],
    stderr_bytes: &[u8],
) -> Result<ExecutionResult, String> {
    let Some(status) = status else {
        return Err(format!("script execution timeout after {timeout_ms}ms"));
    };
    if !status.success() {
        return Err(worker_exit_error(status, stderr_bytes));
    }
    decode_worker_result(stdout_bytes)
}

fn join_worker_output(
    stdout_handle: thread::JoinHandle<Vec<u8>>,
    stderr_handle: thread::JoinHandle<Vec<u8>>,
) -> (Vec<u8>, Vec<u8>) {
    let stdout_bytes = stdout_handle.join().unwrap_or_default();
    let stderr_bytes = stderr_handle.join().unwrap_or_default();
    (stdout_bytes, stderr_bytes)
}

fn spawn_worker_process(worker_request: &ScriptWorkerRequest) -> Result<WorkerProcess, String> {
    let mut child = build_worker_command()?
        .spawn()
        .map_err(|err| format!("spawn sandbox worker failed: {err}"))?;
    write_worker_request(&mut child, worker_request)?;

    let stdout = take_child_pipe(child.stdout.take(), "stdout", &mut child)?;
    let stderr = take_child_pipe(child.stderr.take(), "stderr", &mut child)?;

    Ok(WorkerProcess {
        child,
        stdout_handle: spawn_pipe_reader(stdout),
        stderr_handle: spawn_pipe_reader(stderr),
    })
}

fn build_worker_command() -> Result<Command, String> {
    let executable = std::env::current_exe()
        .map_err(|err| format!("resolve current executable failed: {err}"))?;
    let mut command = Command::new(executable);
    command
        .arg("--sandbox-worker")
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    Ok(command)
}

fn write_worker_request(
    child: &mut Child,
    worker_request: &ScriptWorkerRequest,
) -> Result<(), String> {
    let mut stdin = child
        .stdin
        .take()
        .ok_or_else(|| "sandbox worker stdin is not available".to_string())?;
    if let Err(err) = serde_json::to_writer(&mut stdin, worker_request) {
        terminate_child(child);
        return Err(format!("encode sandbox worker request failed: {err}"));
    }
    if let Err(err) = stdin.write_all(b"\n") {
        terminate_child(child);
        return Err(format!("flush sandbox worker request failed: {err}"));
    }
    Ok(())
}

fn take_child_pipe<T>(pipe: Option<T>, stream_name: &str, child: &mut Child) -> Result<T, String> {
    match pipe {
        Some(pipe) => Ok(pipe),
        None => {
            terminate_child(child);
            Err(format!("sandbox worker {stream_name} is not available"))
        }
    }
}

fn terminate_child(child: &mut Child) {
    let _ = child.kill();
    let _ = child.wait();
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
            Ok(None) if Instant::now() < deadline => {
                thread::sleep(Duration::from_millis(10));
            }
            Ok(None) => {
                terminate_child(child);
                return Ok(None);
            }
            Err(err) => {
                terminate_child(child);
                return Err(format!("wait sandbox worker failed: {err}"));
            }
        }
    }
}

#[cfg(unix)]
pub(super) fn apply_memory_limit(max_memory_mb: u64) -> Result<(), String> {
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
pub(super) fn apply_memory_limit(_max_memory_mb: u64) -> Result<(), String> {
    Ok(())
}
