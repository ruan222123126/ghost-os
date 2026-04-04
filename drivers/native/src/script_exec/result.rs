use std::io::{self, Write};
#[cfg(feature = "python-sandbox")]
use std::process::ExitStatus;

use crate::sandbox::ExecutionResult;

#[cfg(feature = "python-sandbox")]
pub(super) fn decode_worker_result(stdout_bytes: &[u8]) -> Result<ExecutionResult, String> {
    serde_json::from_slice::<ExecutionResult>(stdout_bytes).map_err(|err| {
        let stdout_text = String::from_utf8_lossy(stdout_bytes);
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

#[cfg(feature = "python-sandbox")]
pub(super) fn worker_exit_error(status: ExitStatus, stderr_bytes: &[u8]) -> String {
    let stderr_text = String::from_utf8_lossy(stderr_bytes);
    let stderr_line = stderr_text.lines().next().unwrap_or("").trim();
    if stderr_line.is_empty() {
        return format!("sandbox worker exited with status {status}");
    }
    format!("sandbox worker exited with status {status}: {stderr_line}")
}

pub(super) fn emit_worker_result(result: ExecutionResult) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&result) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}
