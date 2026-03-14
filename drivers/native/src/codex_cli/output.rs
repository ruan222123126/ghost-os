use regex::Regex;
use serde_json::Value;
use std::io::{BufRead, BufReader, Read};
use std::sync::{Arc, Mutex, OnceLock};
use std::thread;

use super::MAX_OUTPUT_BUFFER_CHARS;

pub(super) type SharedOutputBuffer = Arc<Mutex<OutputBuffer>>;
pub(super) type SharedSessionId = Arc<Mutex<Option<String>>>;

#[derive(Default)]
pub(super) struct OutputBuffer {
    data: String,
}

impl OutputBuffer {
    pub(super) fn push(&mut self, text: &str) {
        if text.is_empty() {
            return;
        }
        self.data.push_str(text);
        if self.data.chars().count() > MAX_OUTPUT_BUFFER_CHARS {
            self.data = trim_to_last_chars(&self.data, MAX_OUTPUT_BUFFER_CHARS);
        }
    }

    fn snapshot(&self) -> String {
        self.data.clone()
    }
}

pub(super) fn new_shared_output() -> SharedOutputBuffer {
    Arc::new(Mutex::new(OutputBuffer::default()))
}

pub(super) fn new_session_id_slot() -> SharedSessionId {
    Arc::new(Mutex::new(None))
}

pub(super) fn output_tail(output: &SharedOutputBuffer, max_chars: usize) -> String {
    let snapshot = match output.lock() {
        Ok(buffer) => buffer.snapshot(),
        Err(_) => String::new(),
    };
    trim_to_last_chars(&snapshot, max_chars)
}

pub(super) fn session_id_value(session_id: &SharedSessionId) -> Option<String> {
    session_id.lock().ok().and_then(|guard| guard.clone())
}

pub(super) fn spawn_output_reader<R: Read + Send + 'static>(
    reader: Option<R>,
    output: SharedOutputBuffer,
    session_id: SharedSessionId,
    prefix: &str,
) {
    let Some(reader) = reader else {
        return;
    };
    let prefix = prefix.to_string();
    thread::spawn(move || read_output_lines(reader, output, session_id, prefix));
}

fn read_output_lines<R: Read>(
    reader: R,
    output: SharedOutputBuffer,
    session_id: SharedSessionId,
    prefix: String,
) {
    let mut reader = BufReader::new(reader);
    let mut line = String::new();
    loop {
        line.clear();
        match reader.read_line(&mut line) {
            Ok(0) => break,
            Ok(_) => record_output_line(&line, &output, &session_id, &prefix),
            Err(_) => break,
        }
    }
}

fn record_output_line(
    line: &str,
    output: &SharedOutputBuffer,
    session_id: &SharedSessionId,
    prefix: &str,
) {
    let trimmed = line.trim_end_matches(&['\r', '\n'][..]);
    update_session_id(trimmed, session_id);
    let Ok(mut buffer) = output.lock() else {
        return;
    };
    buffer.push(&format!("{prefix}{line}"));
}

fn update_session_id(line: &str, session_id: &SharedSessionId) {
    let Some(parsed) = parse_session_id(line) else {
        return;
    };
    let Ok(mut guard) = session_id.lock() else {
        return;
    };
    if guard.is_none() {
        *guard = Some(parsed);
    }
}

fn parse_session_id(line: &str) -> Option<String> {
    let trimmed = line.trim();
    if trimmed.is_empty() {
        return None;
    }
    parse_json_session_id(trimmed).or_else(|| parse_text_session_id(trimmed))
}

fn parse_json_session_id(trimmed: &str) -> Option<String> {
    if !trimmed.starts_with('{') || !trimmed.ends_with('}') {
        return None;
    }
    let value = serde_json::from_str::<Value>(trimmed).ok()?;
    ["session_id", "sessionId"]
        .into_iter()
        .find_map(|field| extract_session_id(&value, field))
}

fn extract_session_id(value: &Value, field: &str) -> Option<String> {
    let id = value.get(field).and_then(Value::as_str)?.trim();
    if id.is_empty() {
        return None;
    }
    Some(id.to_string())
}

fn parse_text_session_id(trimmed: &str) -> Option<String> {
    for regex in session_id_patterns() {
        let Some(caps) = regex.captures(trimmed) else {
            continue;
        };
        let Some(id) = caps.get(1) else {
            continue;
        };
        let value = id.as_str().trim();
        if !value.is_empty() {
            return Some(value.to_string());
        }
    }
    None
}

fn session_id_patterns() -> &'static [Regex; 3] {
    static PATTERNS: OnceLock<[Regex; 3]> = OnceLock::new();
    PATTERNS.get_or_init(|| {
        [
            Regex::new(r"(?i)session[_\s-]*id[:=]\s*([A-Za-z0-9_-]{6,})").unwrap(),
            Regex::new(r"(?i)session\s+id[:=]\s*([A-Za-z0-9_-]{6,})").unwrap(),
            Regex::new(r"(?i)session\s+([0-9a-fA-F-]{8,})").unwrap(),
        ]
    })
}

pub(super) fn trim_to_last_chars(input: &str, max_chars: usize) -> String {
    if max_chars == 0 {
        return String::new();
    }
    let total = input.chars().count();
    if total <= max_chars {
        return input.to_string();
    }
    let skip = total - max_chars;
    input.chars().skip(skip).collect()
}
