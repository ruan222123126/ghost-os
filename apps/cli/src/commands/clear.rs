use std::env;
use std::io::{self, Write};
use std::process::Command;

use anyhow::Result;

pub(crate) fn clear_screen() -> Result<()> {
    let mut stdout = io::stdout();
    clear_screen_with(prefer_ansi_clear(), run_system_clear, &mut stdout)
}

pub(crate) fn clear_screen_with(
    prefer_ansi: bool,
    run_system_clear: impl FnOnce() -> io::Result<bool>,
    out: &mut dyn Write,
) -> Result<()> {
    if prefer_ansi {
        write_ansi_clear(out)?;
        return Ok(());
    }

    match run_system_clear() {
        Ok(true) => Ok(()),
        _ => {
            write_ansi_clear(out)?;
            Ok(())
        }
    }
}

fn run_system_clear() -> io::Result<bool> {
    let status = if cfg!(windows) {
        Command::new("cmd").args(["/C", "cls"]).status()
    } else {
        Command::new("clear").status()
    };
    status.map(|value| value.success())
}

fn write_ansi_clear(out: &mut dyn Write) -> io::Result<()> {
    // ANSI: clear screen + move cursor to top-left.
    write!(out, "\x1B[2J\x1B[H")?;
    out.flush()
}

fn prefer_ansi_clear() -> bool {
    if cfg!(windows) {
        return env::var_os("WT_SESSION").is_some()
            || env::var_os("ANSICON").is_some()
            || env::var("TERM").map(|term| term != "dumb").unwrap_or(false);
    }
    env::var("TERM").map(|term| term != "dumb").unwrap_or(true)
}

#[cfg(test)]
mod tests {
    use std::cell::Cell;
    use std::io;

    use super::clear_screen_with;

    #[test]
    fn clear_screen_falls_back_to_ansi_when_system_clear_fails() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Ok(false)
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }

    #[test]
    fn clear_screen_uses_system_clear_when_ansi_not_preferred() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Ok(true)
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert!(buf.is_empty());
    }

    #[test]
    fn clear_screen_uses_ansi_first_when_preferred() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            true,
            || {
                called.set(true);
                Ok(true)
            },
            &mut buf,
        )
        .unwrap();

        assert!(!called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }

    #[test]
    fn clear_screen_falls_back_to_ansi_when_system_clear_errors() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Err(io::Error::other("failed"))
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }
}
