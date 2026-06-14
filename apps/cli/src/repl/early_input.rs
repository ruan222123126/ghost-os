use anyhow::Result;
use std::io::IsTerminal;
use std::time::Duration;

const CAPTURE_WINDOW_MS: u64 = 180;
const POLL_INTERVAL_MS: u64 = 12;

pub(super) fn capture_first_line_prefill() -> Result<Option<String>> {
    if !std::io::stdin().is_terminal() {
        return Ok(None);
    }

    #[cfg(unix)]
    {
        capture_tty_prefill(
            Duration::from_millis(CAPTURE_WINDOW_MS),
            Duration::from_millis(POLL_INTERVAL_MS),
        )
    }

    #[cfg(not(unix))]
    {
        Ok(None)
    }
}

#[cfg(unix)]
mod platform {
    use super::{CapturedInput, Result};
    use anyhow::Context;
    use std::io;
    use std::mem::MaybeUninit;
    use std::os::fd::AsRawFd;
    use std::os::fd::RawFd;
    use std::thread;
    use std::time::{Duration, Instant};

    const READ_BUFFER_SIZE: usize = 128;

    pub(super) fn capture_tty_prefill(window: Duration, poll: Duration) -> Result<Option<String>> {
        let stdin = io::stdin();
        let fd = stdin.as_raw_fd();
        let _guard = RawCaptureGuard::enter(fd)?;
        let deadline = Instant::now() + window;
        let mut captured = CapturedInput::default();

        loop {
            let status = read_pending(fd, &mut captured)?;
            if status.terminated || Instant::now() >= deadline {
                break;
            }
            if status.read_any {
                continue;
            }

            let now = Instant::now();
            if now >= deadline {
                break;
            }
            thread::sleep((deadline - now).min(poll));
        }

        Ok(captured.finish())
    }

    struct ReadStatus {
        read_any: bool,
        terminated: bool,
    }

    fn read_pending(fd: RawFd, captured: &mut CapturedInput) -> Result<ReadStatus> {
        let mut read_any = false;
        let mut buffer = [0_u8; READ_BUFFER_SIZE];

        loop {
            let count = read_nonblocking(fd, &mut buffer)?;
            if count == 0 {
                break;
            }
            read_any = true;
            captured.push_bytes(&buffer[..count]);
            if captured.is_terminated() {
                break;
            }
        }

        Ok(ReadStatus {
            read_any,
            terminated: captured.is_terminated(),
        })
    }

    fn read_nonblocking(fd: RawFd, buffer: &mut [u8]) -> Result<usize> {
        loop {
            // SAFETY: `buffer` is valid for writes of `buffer.len()` bytes, and
            // `fd` is an owned process file descriptor.
            let read_count = unsafe { libc::read(fd, buffer.as_mut_ptr().cast(), buffer.len()) };
            if read_count >= 0 {
                return Ok(read_count as usize);
            }

            let err = io::Error::last_os_error();
            if err.kind() == io::ErrorKind::WouldBlock {
                return Ok(0);
            }
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err).context("failed to read early REPL input");
        }
    }

    struct RawCaptureGuard {
        fd: RawFd,
        original_mode: libc::termios,
        original_flags: libc::c_int,
    }

    impl RawCaptureGuard {
        fn enter(fd: RawFd) -> Result<Self> {
            let original_mode = read_termios(fd)?;
            let original_flags = read_fd_flags(fd)?;
            set_fd_flags(fd, original_flags | libc::O_NONBLOCK)?;

            let mut capture_mode = original_mode;
            capture_mode.c_lflag &= !(libc::ICANON | libc::ECHO);
            capture_mode.c_cc[libc::VMIN] = 0;
            capture_mode.c_cc[libc::VTIME] = 0;
            if let Err(err) = set_termios(fd, &capture_mode) {
                let _ = set_fd_flags(fd, original_flags);
                return Err(err);
            }

            Ok(Self {
                fd,
                original_mode,
                original_flags,
            })
        }
    }

    impl Drop for RawCaptureGuard {
        fn drop(&mut self) {
            let _ = set_termios(self.fd, &self.original_mode);
            let _ = set_fd_flags(self.fd, self.original_flags);
        }
    }

    fn read_termios(fd: RawFd) -> Result<libc::termios> {
        let mut state = MaybeUninit::<libc::termios>::uninit();
        // SAFETY: `state` points to writable memory for `termios`.
        let status = unsafe { libc::tcgetattr(fd, state.as_mut_ptr()) };
        if status == 0 {
            // SAFETY: `tcgetattr` initialized `state` on success.
            return Ok(unsafe { state.assume_init() });
        }
        Err(io::Error::last_os_error()).context("failed to read tty mode")
    }

    fn set_termios(fd: RawFd, termios: &libc::termios) -> Result<()> {
        // SAFETY: `termios` points to a valid termios structure.
        let status = unsafe { libc::tcsetattr(fd, libc::TCSANOW, termios) };
        if status == 0 {
            return Ok(());
        }
        Err(io::Error::last_os_error()).context("failed to update tty mode")
    }

    fn read_fd_flags(fd: RawFd) -> Result<libc::c_int> {
        // SAFETY: `fcntl` is called with a valid fd and `F_GETFL` command.
        let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
        if flags >= 0 {
            return Ok(flags);
        }
        Err(io::Error::last_os_error()).context("failed to read stdin fd flags")
    }

    fn set_fd_flags(fd: RawFd, flags: libc::c_int) -> Result<()> {
        // SAFETY: `fcntl` is called with a valid fd and `F_SETFL` command.
        let status = unsafe { libc::fcntl(fd, libc::F_SETFL, flags) };
        if status == 0 {
            return Ok(());
        }
        Err(io::Error::last_os_error()).context("failed to update stdin fd flags")
    }
}

#[cfg(unix)]
use platform::capture_tty_prefill;

#[derive(Clone, Copy, Default)]
enum EscapeState {
    #[default]
    None,
    Esc,
    Csi,
}

#[derive(Default)]
struct CapturedInput {
    text: String,
    utf8_pending: Vec<u8>,
    escape_state: EscapeState,
    terminated: bool,
}

impl CapturedInput {
    fn push_bytes(&mut self, bytes: &[u8]) {
        for byte in bytes {
            if self.terminated {
                return;
            }
            self.push_byte(*byte);
        }
    }

    fn push_byte(&mut self, byte: u8) {
        if !matches!(self.escape_state, EscapeState::None) {
            self.handle_escape_byte(byte);
            return;
        }

        match byte {
            b'\n' | b'\r' => self.terminated = true,
            0x1b => self.escape_state = EscapeState::Esc,
            0x08 | 0x7f => self.handle_backspace(),
            0x00..=0x1f => {}
            _ => self.push_text_byte(byte),
        }
    }

    fn handle_escape_byte(&mut self, byte: u8) {
        self.escape_state = match self.escape_state {
            EscapeState::None => EscapeState::None,
            EscapeState::Esc => {
                if matches!(byte, b'[' | b'O') {
                    EscapeState::Csi
                } else {
                    EscapeState::None
                }
            }
            EscapeState::Csi => {
                if (0x40..=0x7e).contains(&byte) {
                    EscapeState::None
                } else {
                    EscapeState::Csi
                }
            }
        };
    }

    fn handle_backspace(&mut self) {
        if !self.utf8_pending.is_empty() {
            self.utf8_pending.pop();
            return;
        }
        self.text.pop();
    }

    fn push_text_byte(&mut self, byte: u8) {
        self.utf8_pending.push(byte);
        match std::str::from_utf8(&self.utf8_pending) {
            Ok(segment) => {
                self.text.push_str(segment);
                self.utf8_pending.clear();
            }
            Err(err) if err.error_len().is_some() => {
                self.utf8_pending.clear();
            }
            Err(_) => {}
        }
    }

    fn is_terminated(&self) -> bool {
        self.terminated
    }

    fn finish(self) -> Option<String> {
        if self.text.is_empty() {
            return None;
        }
        Some(self.text)
    }
}

#[cfg(test)]
mod tests;
