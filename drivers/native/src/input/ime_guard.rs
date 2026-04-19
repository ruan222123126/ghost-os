#[cfg(target_os = "linux")]
use std::io::ErrorKind;
#[cfg(target_os = "linux")]
use std::process::{Command, Output};

#[cfg(target_os = "linux")]
const FCITX_ACTIVE_STATE: i32 = 2;
#[cfg(target_os = "linux")]
const IBUS_ENGINE_RANK_EXACT_US: i32 = 100;
#[cfg(target_os = "linux")]
const IBUS_ENGINE_RANK_US: i32 = 90;
#[cfg(target_os = "linux")]
const IBUS_ENGINE_RANK_XKB: i32 = 80;

#[cfg(target_os = "linux")]
enum RestoreAction {
    Fcitx5Activate,
    FcitxActivate,
    IBusEngine(String),
}

#[cfg(target_os = "linux")]
pub(crate) struct X11InputMethodGuard {
    restore_action: Option<RestoreAction>,
}

#[cfg(target_os = "linux")]
impl X11InputMethodGuard {
    pub(crate) fn activate() -> Result<Self, String> {
        if let Some(action) = activate_fcitx5()? {
            return Ok(Self {
                restore_action: Some(action),
            });
        }
        if let Some(action) = activate_fcitx()? {
            return Ok(Self {
                restore_action: Some(action),
            });
        }
        if let Some(action) = activate_ibus()? {
            return Ok(Self {
                restore_action: Some(action),
            });
        }
        Ok(Self {
            restore_action: None,
        })
    }

    pub(crate) fn restore(self) -> Result<(), String> {
        let Some(action) = self.restore_action else {
            return Ok(());
        };
        restore_input_method(action)
    }
}

#[cfg(not(target_os = "linux"))]
pub(crate) struct X11InputMethodGuard;

#[cfg(not(target_os = "linux"))]
impl X11InputMethodGuard {
    pub(crate) fn activate() -> Result<Self, String> {
        Ok(Self)
    }

    pub(crate) fn restore(self) -> Result<(), String> {
        Ok(())
    }
}

#[cfg(target_os = "linux")]
fn activate_fcitx5() -> Result<Option<RestoreAction>, String> {
    let state = query_fcitx_state("fcitx5-remote")?;
    if state != Some(FCITX_ACTIVE_STATE) {
        return Ok(None);
    }
    run_simple_command("fcitx5-remote", &["-c"])?;
    Ok(Some(RestoreAction::Fcitx5Activate))
}

#[cfg(target_os = "linux")]
fn activate_fcitx() -> Result<Option<RestoreAction>, String> {
    let state = query_fcitx_state("fcitx-remote")?;
    if state != Some(FCITX_ACTIVE_STATE) {
        return Ok(None);
    }
    run_simple_command("fcitx-remote", &["-c"])?;
    Ok(Some(RestoreAction::FcitxActivate))
}

#[cfg(target_os = "linux")]
fn query_fcitx_state(program: &str) -> Result<Option<i32>, String> {
    let Some(output) = command_output_optional(program, &[])? else {
        return Ok(None);
    };
    if !output.status.success() {
        return Ok(None);
    }
    Ok(parse_fcitx_state_text(stdout_trimmed(&output)))
}

#[cfg(target_os = "linux")]
fn parse_fcitx_state_text(text: &str) -> Option<i32> {
    let trimmed = text.trim();
    if trimmed.is_empty() {
        return None;
    }
    trimmed.parse::<i32>().ok()
}

#[cfg(target_os = "linux")]
fn activate_ibus() -> Result<Option<RestoreAction>, String> {
    let Some(current_engine) = read_ibus_current_engine()? else {
        return Ok(None);
    };
    if current_engine.starts_with("xkb:") {
        return Ok(None);
    }
    let Some(ascii_engine) = pick_ibus_ascii_engine()? else {
        return Err("cannot resolve an ASCII ibus engine to avoid IME conversion".to_string());
    };
    if current_engine == ascii_engine {
        return Ok(None);
    }
    run_simple_command("ibus", &["engine", &ascii_engine])?;
    Ok(Some(RestoreAction::IBusEngine(current_engine)))
}

#[cfg(target_os = "linux")]
fn read_ibus_current_engine() -> Result<Option<String>, String> {
    let Some(output) = command_output_optional("ibus", &["engine"])? else {
        return Ok(None);
    };
    if !output.status.success() {
        return Ok(None);
    }
    let text = stdout_trimmed(&output);
    if text.is_empty() || text == "No engine is set." {
        return Ok(None);
    }
    Ok(Some(text.to_string()))
}

#[cfg(target_os = "linux")]
fn pick_ibus_ascii_engine() -> Result<Option<String>, String> {
    let Some(output) = command_output_optional("ibus", &["list-engine"])? else {
        return Ok(Some("xkb:us::eng".to_string()));
    };
    if !output.status.success() {
        return Ok(Some("xkb:us::eng".to_string()));
    }
    let best = select_best_ibus_ascii_engine(&output.stdout);
    if let Some(engine) = best {
        return Ok(Some(engine));
    }
    Ok(Some("xkb:us::eng".to_string()))
}

#[cfg(target_os = "linux")]
fn select_best_ibus_ascii_engine(stdout: &[u8]) -> Option<String> {
    stdout
        .split(|byte| *byte == b'\n')
        .filter_map(parse_ibus_engine_line)
        .max_by_key(|engine| ibus_ascii_engine_rank(engine))
}

#[cfg(target_os = "linux")]
fn parse_ibus_engine_line(line: &[u8]) -> Option<String> {
    let engine = String::from_utf8_lossy(line).trim().to_string();
    if engine.is_empty() {
        return None;
    }
    if !engine.contains(':') || engine.contains(' ') {
        return None;
    }
    Some(engine)
}

#[cfg(target_os = "linux")]
fn ibus_ascii_engine_rank(engine: &str) -> i32 {
    if engine == "xkb:us::eng" {
        return IBUS_ENGINE_RANK_EXACT_US;
    }
    if engine.starts_with("xkb:us") {
        return IBUS_ENGINE_RANK_US;
    }
    if engine.starts_with("xkb:") {
        return IBUS_ENGINE_RANK_XKB;
    }
    0
}

#[cfg(target_os = "linux")]
fn restore_input_method(action: RestoreAction) -> Result<(), String> {
    match action {
        RestoreAction::Fcitx5Activate => run_simple_command("fcitx5-remote", &["-o"]),
        RestoreAction::FcitxActivate => run_simple_command("fcitx-remote", &["-o"]),
        RestoreAction::IBusEngine(engine) => run_simple_command("ibus", &["engine", &engine]),
    }
}

#[cfg(target_os = "linux")]
fn command_output_optional(program: &str, args: &[&str]) -> Result<Option<Output>, String> {
    match Command::new(program).args(args).output() {
        Ok(output) => Ok(Some(output)),
        Err(err) if err.kind() == ErrorKind::NotFound => Ok(None),
        Err(err) => Err(format!("spawn {program} failed: {err}")),
    }
}

#[cfg(target_os = "linux")]
fn run_simple_command(program: &str, args: &[&str]) -> Result<(), String> {
    let output = Command::new(program)
        .args(args)
        .output()
        .map_err(|err| format!("spawn {program} failed: {err}"))?;
    if output.status.success() {
        return Ok(());
    }
    let detail = first_non_empty_line(&output).unwrap_or("no error details");
    Err(format!(
        "{program} failed with status {}: {detail}",
        output.status
    ))
}

#[cfg(target_os = "linux")]
fn stdout_trimmed(output: &Output) -> &str {
    std::str::from_utf8(&output.stdout)
        .map(str::trim)
        .unwrap_or("")
}

#[cfg(target_os = "linux")]
fn first_non_empty_line(output: &Output) -> Option<&str> {
    for line in std::str::from_utf8(&output.stderr).unwrap_or("").lines() {
        let trimmed = line.trim();
        if !trimmed.is_empty() {
            return Some(trimmed);
        }
    }
    for line in std::str::from_utf8(&output.stdout).unwrap_or("").lines() {
        let trimmed = line.trim();
        if !trimmed.is_empty() {
            return Some(trimmed);
        }
    }
    None
}

#[cfg(test)]
#[path = "ime_guard_tests.rs"]
mod tests;
