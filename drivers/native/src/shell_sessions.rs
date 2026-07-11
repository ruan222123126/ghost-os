use crate::sandbox::shell_tools::truncate_shell_output;
use libc::{F_GETFL, F_SETFL, O_NONBLOCK, fcntl};
use std::collections::HashMap;
use std::io::{self, ErrorKind, Read, Write};
use std::os::fd::AsRawFd;
use std::process::{Child, ChildStderr, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::{Mutex, OnceLock};
use std::thread;
use std::time::{Duration, Instant};

const DEFAULT_INTERACTIVE_YIELD_MS: u64 = 100;
const MAX_INTERACTIVE_YIELD_MS: u64 = 60_000;
const SESSION_TTL: Duration = Duration::from_secs(10 * 60);
const MAX_SHELL_SESSIONS: usize = 16;
const READ_BUFFER_BYTES: usize = 4096;
const EXIT_MARKER_PREFIX: &str = "__GHOST_EXIT_CODE__:";

pub(crate) struct ShellSessionRequest {
    pub(crate) command: String,
    pub(crate) session_id: Option<String>,
    pub(crate) tty: bool,
    pub(crate) yield_time_ms: Option<u64>,
    pub(crate) max_output_chars: usize,
}

pub(crate) struct ShellSessionResponse {
    pub(crate) session_id: String,
    pub(crate) stdout: String,
    pub(crate) stderr: String,
    pub(crate) command_exit_code: Option<i32>,
    pub(crate) running: bool,
    pub(crate) shell_exit_code: Option<i32>,
    pub(crate) reused: bool,
}

struct ShellSessionPool {
    next_id: u64,
    sessions: HashMap<String, ShellSession>,
}

struct ShellSession {
    child: Child,
    stdin: ChildStdin,
    stdout: ChildStdout,
    stderr: ChildStderr,
    stdout_buffer: String,
    stderr_buffer: String,
    last_used_at: Instant,
}

struct CollectedOutput {
    stdout: String,
    stderr: String,
    command_exit_code: Option<i32>,
}

static SHELL_SESSION_POOL: OnceLock<Mutex<ShellSessionPool>> = OnceLock::new();

pub(crate) fn run_shell_session(
    request: ShellSessionRequest,
) -> Result<ShellSessionResponse, String> {
    if cfg!(target_os = "windows") {
        return Err("interactive shell sessions are not supported on windows".to_string());
    }
    if request.tty {
        return Err("interactive shell sessions do not support tty=true yet".to_string());
    }

    let yield_ms = resolve_yield_ms(request.yield_time_ms)?;
    let pool = shell_session_pool();
    let mut guard = pool
        .lock()
        .map_err(|_| "shell session pool lock poisoned".to_string())?;
    guard.cleanup_stale();
    guard.execute(request, Duration::from_millis(yield_ms))
}

fn shell_session_pool() -> &'static Mutex<ShellSessionPool> {
    SHELL_SESSION_POOL.get_or_init(|| {
        Mutex::new(ShellSessionPool {
            next_id: 0,
            sessions: HashMap::new(),
        })
    })
}

fn resolve_yield_ms(value: Option<u64>) -> Result<u64, String> {
    let resolved = value.unwrap_or(DEFAULT_INTERACTIVE_YIELD_MS);
    if resolved == 0 {
        return Err("yield_time_ms must be >= 1".to_string());
    }
    Ok(resolved.min(MAX_INTERACTIVE_YIELD_MS))
}

impl ShellSessionPool {
    fn execute(
        &mut self,
        request: ShellSessionRequest,
        yield_window: Duration,
    ) -> Result<ShellSessionResponse, String> {
        let (session_id, reused) = self.ensure_session(request.session_id.as_deref())?;
        let session = self
            .sessions
            .get_mut(&session_id)
            .ok_or_else(|| "shell session disappeared".to_string())?;
        session.last_used_at = Instant::now();

        write_command(&mut session.stdin, &request.command)?;
        let output = collect_output(session, yield_window, request.max_output_chars)?;
        let (running, shell_exit_code) = poll_session_status(&mut session.child)?;

        let response = ShellSessionResponse {
            session_id: session_id.clone(),
            stdout: output.stdout,
            stderr: output.stderr,
            command_exit_code: output.command_exit_code,
            running,
            shell_exit_code,
            reused,
        };
        if !running {
            self.sessions.remove(&session_id);
        }
        Ok(response)
    }

    fn ensure_session(&mut self, requested_id: Option<&str>) -> Result<(String, bool), String> {
        let Some(raw_id) = requested_id else {
            return self.create_session();
        };
        let session_id = raw_id.trim();
        if session_id.is_empty() {
            return Err("session_id must be a non-empty string".to_string());
        }
        let Some(session) = self.sessions.get_mut(session_id) else {
            return Err(format!("unknown session_id: {session_id}"));
        };
        let (running, _) = poll_session_status(&mut session.child)?;
        if !running {
            self.sessions.remove(session_id);
            return Err(format!("unknown session_id: {session_id}"));
        }
        session.last_used_at = Instant::now();
        Ok((session_id.to_string(), true))
    }

    fn create_session(&mut self) -> Result<(String, bool), String> {
        if self.sessions.len() >= MAX_SHELL_SESSIONS {
            return Err(format!(
                "shell session limit reached: max {}",
                MAX_SHELL_SESSIONS
            ));
        }

        let session_id = self.next_session_id();
        self.sessions
            .insert(session_id.clone(), spawn_shell_session()?);
        Ok((session_id, false))
    }

    fn cleanup_stale(&mut self) {
        let now = Instant::now();
        self.sessions.retain(|_, session| {
            let stale = now.duration_since(session.last_used_at) > SESSION_TTL;
            if stale {
                terminate_session(session);
            }
            !stale
        });
    }

    fn next_session_id(&mut self) -> String {
        self.next_id += 1;
        format!("bash-{}", self.next_id)
    }
}

fn spawn_shell_session() -> Result<ShellSession, String> {
    let mut command = Command::new("bash");
    command
        .arg("-i")
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    let mut child = command
        .spawn()
        .map_err(|err| format!("failed to spawn shell session: {err}"))?;

    let stdin = child
        .stdin
        .take()
        .ok_or_else(|| "shell session stdin is not available".to_string())?;
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| "shell session stdout is not available".to_string())?;
    let stderr = child
        .stderr
        .take()
        .ok_or_else(|| "shell session stderr is not available".to_string())?;

    set_nonblocking(&stdout)
        .map_err(|err| format!("configure stdout nonblocking failed: {err}"))?;
    set_nonblocking(&stderr)
        .map_err(|err| format!("configure stderr nonblocking failed: {err}"))?;

    Ok(ShellSession {
        child,
        stdin,
        stdout,
        stderr,
        stdout_buffer: String::new(),
        stderr_buffer: String::new(),
        last_used_at: Instant::now(),
    })
}

fn set_nonblocking<T: AsRawFd>(stream: &T) -> io::Result<()> {
    let fd = stream.as_raw_fd();
    // SAFETY: fcntl is called with valid pipe descriptors from child stdout/stderr.
    let flags = unsafe { fcntl(fd, F_GETFL) };
    if flags < 0 {
        return Err(io::Error::last_os_error());
    }
    // SAFETY: fcntl F_SETFL updates flags for the same valid descriptor.
    let result = unsafe { fcntl(fd, F_SETFL, flags | O_NONBLOCK) };
    if result < 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

fn write_command(stdin: &mut ChildStdin, command: &str) -> Result<(), String> {
    stdin
        .write_all(command.as_bytes())
        .map_err(|err| format!("write shell command failed: {err}"))?;
    stdin
        .write_all(b"\n")
        .map_err(|err| format!("write shell newline failed: {err}"))?;
    stdin
        .write_all(format!("printf '{}%s\\n' \"$?\"\n", EXIT_MARKER_PREFIX).as_bytes())
        .map_err(|err| format!("write shell exit marker failed: {err}"))?;
    stdin
        .flush()
        .map_err(|err| format!("flush shell stdin failed: {err}"))?;
    Ok(())
}

fn collect_output(
    session: &mut ShellSession,
    yield_window: Duration,
    max_output_chars: usize,
) -> Result<CollectedOutput, String> {
    let deadline = Instant::now() + yield_window;
    let mut command_exit_code = None;

    loop {
        session
            .stdout_buffer
            .push_str(&read_available(&mut session.stdout)?);
        session
            .stderr_buffer
            .push_str(&read_available(&mut session.stderr)?);
        command_exit_code = extract_exit_code(&mut session.stdout_buffer, command_exit_code);
        if command_exit_code.is_some() || Instant::now() >= deadline {
            break;
        }
        if !poll_session_status(&mut session.child)?.0 {
            break;
        }
        thread::sleep(Duration::from_millis(10));
    }

    Ok(CollectedOutput {
        stdout: truncate_shell_output(&take_buffer(&mut session.stdout_buffer), max_output_chars),
        stderr: truncate_shell_output(&take_buffer(&mut session.stderr_buffer), max_output_chars),
        command_exit_code,
    })
}

fn read_available<T: Read>(reader: &mut T) -> Result<String, String> {
    let mut output = Vec::new();
    let mut buffer = [0_u8; READ_BUFFER_BYTES];
    loop {
        match reader.read(&mut buffer) {
            Ok(0) => break,
            Ok(count) => output.extend_from_slice(&buffer[..count]),
            Err(err) if err.kind() == ErrorKind::WouldBlock => break,
            Err(err) => return Err(format!("read shell stream failed: {err}")),
        }
    }
    Ok(String::from_utf8_lossy(&output).to_string())
}

fn poll_session_status(child: &mut Child) -> Result<(bool, Option<i32>), String> {
    match child.try_wait() {
        Ok(Some(status)) => Ok((false, status.code())),
        Ok(None) => Ok((true, None)),
        Err(err) => Err(format!("check shell session status failed: {err}")),
    }
}

fn take_buffer(buffer: &mut String) -> String {
    let output = buffer.clone();
    buffer.clear();
    output
}

fn extract_exit_code(buffer: &mut String, current: Option<i32>) -> Option<i32> {
    if current.is_some() {
        return current;
    }
    let marker_index = buffer.find(EXIT_MARKER_PREFIX)?;
    let line_end = buffer[marker_index..].find('\n')?;
    let absolute_end = marker_index + line_end;
    let raw = &buffer[(marker_index + EXIT_MARKER_PREFIX.len())..absolute_end];
    let parsed = raw.trim().parse::<i32>().ok();
    buffer.replace_range(marker_index..=absolute_end, "");
    parsed
}

fn terminate_session(session: &mut ShellSession) {
    let _ = session.child.kill();
    let _ = session.child.wait();
}

#[cfg(test)]
mod tests {
    use super::{MAX_INTERACTIVE_YIELD_MS, extract_exit_code, resolve_yield_ms};

    #[test]
    fn resolve_yield_ms_defaults_and_clamps() {
        assert_eq!(resolve_yield_ms(None).expect("default"), 100);
        assert_eq!(
            resolve_yield_ms(Some(MAX_INTERACTIVE_YIELD_MS + 100)).expect("clamped"),
            MAX_INTERACTIVE_YIELD_MS
        );
    }

    #[test]
    fn resolve_yield_ms_rejects_zero() {
        let err = resolve_yield_ms(Some(0)).expect_err("must reject");
        assert!(err.contains("yield_time_ms"));
    }

    #[test]
    fn extract_exit_code_parses_and_removes_marker() {
        let mut output = "hello\n__GHOST_EXIT_CODE__:7\n".to_string();
        let code = extract_exit_code(&mut output, None);
        assert_eq!(code, Some(7));
        assert_eq!(output, "hello\n");
    }
}
