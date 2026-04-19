use std::process::Command;

const XDOTOOL_TYPE_DELAY_MS: &str = "12";
const XDOTOOL_UNICODE_KEY_PREFIX: &str = "U";

#[derive(Debug, PartialEq, Eq)]
pub(super) enum XdotoolTextToken {
    AsciiChunk(String),
    Keysym(String),
}

pub(super) fn active_window_id_for_xdotool() -> Result<String, String> {
    let output = Command::new("xdotool")
        .arg("getactivewindow")
        .output()
        .map_err(|err| format!("spawn xdotool failed: {err}"))?;
    if !output.status.success() {
        return Err(format!(
            "xdotool exited with status {}; ensure xdotool is installed and graphical session is active",
            output.status
        ));
    }
    parse_active_window_id_output(&output.stdout)
}

pub(super) fn parse_active_window_id_output(stdout: &[u8]) -> Result<String, String> {
    let window_id = String::from_utf8_lossy(stdout).trim().to_string();
    if window_id.is_empty() {
        return Err("xdotool getactivewindow returned empty output".to_string());
    }
    Ok(window_id)
}

pub(super) fn type_lines_with_xdotool(
    window_id: &str,
    text: &str,
    submit: bool,
) -> Result<(), String> {
    let lines: Vec<&str> = text.split('\n').collect();
    for (index, segment) in lines.iter().enumerate() {
        if !segment.is_empty() {
            type_text_with_xdotool(window_id, segment)?;
        }
        if index + 1 < lines.len() || submit {
            trigger_xdotool_key(window_id, "Return")?;
        }
    }
    Ok(())
}

fn type_text_with_xdotool(window_id: &str, text: &str) -> Result<(), String> {
    for token in tokenize_text_for_xdotool(text) {
        match token {
            XdotoolTextToken::AsciiChunk(chunk) => {
                type_ascii_chunk_with_xdotool(window_id, &chunk)?
            }
            XdotoolTextToken::Keysym(keysym) => trigger_xdotool_key(window_id, &keysym)?,
        }
    }
    Ok(())
}

fn type_ascii_chunk_with_xdotool(window_id: &str, chunk: &str) -> Result<(), String> {
    let mut command = Command::new("xdotool");
    command.args([
        "type",
        "--window",
        window_id,
        "--clearmodifiers",
        "--delay",
        XDOTOOL_TYPE_DELAY_MS,
        "--",
        chunk,
    ]);
    run_xdotool(command)
}

pub(super) fn tokenize_text_for_xdotool(text: &str) -> Vec<XdotoolTextToken> {
    let mut tokens = Vec::new();
    let mut ascii_chunk = String::new();
    for ch in text.chars() {
        if is_ascii_typing_char(ch) {
            ascii_chunk.push(ch);
            continue;
        }
        push_ascii_chunk_if_present(&mut tokens, &mut ascii_chunk);
        tokens.push(XdotoolTextToken::Keysym(unicode_keysym(ch)));
    }
    push_ascii_chunk_if_present(&mut tokens, &mut ascii_chunk);
    tokens
}

fn is_ascii_typing_char(ch: char) -> bool {
    ch.is_ascii_graphic() || ch == ' ' || ch == '\t'
}

fn push_ascii_chunk_if_present(tokens: &mut Vec<XdotoolTextToken>, ascii_chunk: &mut String) {
    if ascii_chunk.is_empty() {
        return;
    }
    tokens.push(XdotoolTextToken::AsciiChunk(std::mem::take(ascii_chunk)));
}

fn unicode_keysym(ch: char) -> String {
    format!("{XDOTOOL_UNICODE_KEY_PREFIX}{:X}", ch as u32)
}

fn trigger_xdotool_key(window_id: &str, keysym: &str) -> Result<(), String> {
    let mut command = Command::new("xdotool");
    command.args(["key", "--window", window_id, "--clearmodifiers", keysym]);
    run_xdotool(command)
}

fn run_xdotool(mut command: Command) -> Result<(), String> {
    let status = command
        .status()
        .map_err(|err| format!("spawn xdotool failed: {err}"))?;
    if status.success() {
        return Ok(());
    }
    Err(format!(
        "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
    ))
}
