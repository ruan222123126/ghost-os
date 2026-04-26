use base64::Engine;
use base64::engine::general_purpose::STANDARD as BASE64_STANDARD;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::process::Command;

use crate::sandbox::SandboxConfig;
use crate::sandbox::path_policy::resolve_read_path;

pub(crate) const DEFAULT_SEARCH_MAX_RESULTS: usize = 50;
const RG_EXIT_MATCH: i32 = 0;
const RG_EXIT_NO_MATCH: i32 = 1;

#[derive(Debug, Clone, Eq, PartialEq, Serialize)]
pub(crate) struct SearchMatchOutput {
    pub(crate) path: String,
    pub(crate) line: usize,
    pub(crate) text: String,
}

pub(crate) fn search_files_impl(
    config: &SandboxConfig,
    query: &str,
    path: &str,
    max_results: usize,
) -> Result<Vec<SearchMatchOutput>, String> {
    let query = normalize_query(query)?;
    validate_max_results(max_results)?;
    let root = resolve_search_root(config, path)?;
    let mut matches = run_ripgrep_search(query, &root)?;
    sort_matches(&mut matches);
    if matches.len() > max_results {
        matches.truncate(max_results);
    }
    Ok(matches)
}

fn normalize_query(query: &str) -> Result<&str, String> {
    let query = query.trim();
    if query.is_empty() {
        return Err("query is required".to_string());
    }
    Ok(query)
}

fn validate_max_results(max_results: usize) -> Result<(), String> {
    if max_results == 0 {
        return Err("max_results must be >= 1".to_string());
    }
    Ok(())
}

fn resolve_search_root(config: &SandboxConfig, path: &str) -> Result<PathBuf, String> {
    let target = normalize_search_path(path);
    let canonical = resolve_read_path(&target, config)?;
    if !canonical.is_dir() {
        return Err(format!("path is not a directory: {}", canonical.display()));
    }
    Ok(canonical)
}

fn normalize_search_path(path: &str) -> String {
    let path = path.trim();
    if path.is_empty() {
        return ".".to_string();
    }
    path.to_string()
}

fn run_ripgrep_search(query: &str, root: &Path) -> Result<Vec<SearchMatchOutput>, String> {
    let output = Command::new("rg")
        .arg("--json")
        .arg("--fixed-strings")
        .arg("--line-number")
        .arg("--no-heading")
        .arg(query)
        .arg(root)
        .output()
        .map_err(format_ripgrep_spawn_error)?;

    if !is_ripgrep_ok_status(output.status.code()) {
        return Err(format_ripgrep_failure(output.status.code(), &output.stderr));
    }
    parse_ripgrep_matches(&output.stdout)
}

fn format_ripgrep_spawn_error(err: std::io::Error) -> String {
    if err.kind() == std::io::ErrorKind::NotFound {
        return "ripgrep (rg) is required but was not found in PATH".to_string();
    }
    format!("failed to execute ripgrep: {err}")
}

fn is_ripgrep_ok_status(code: Option<i32>) -> bool {
    matches!(code, Some(RG_EXIT_MATCH) | Some(RG_EXIT_NO_MATCH))
}

fn format_ripgrep_failure(code: Option<i32>, stderr: &[u8]) -> String {
    let stderr_text = String::from_utf8_lossy(stderr).trim().to_string();
    if stderr_text.is_empty() {
        return format!("ripgrep failed with status {:?}", code);
    }
    format!("ripgrep failed: {stderr_text}")
}

fn parse_ripgrep_matches(stdout: &[u8]) -> Result<Vec<SearchMatchOutput>, String> {
    let mut matches = Vec::new();
    for raw in stdout.split(|byte| *byte == b'\n') {
        if raw.is_empty() {
            continue;
        }
        let text = std::str::from_utf8(raw)
            .map_err(|err| format!("invalid ripgrep json output encoding: {err}"))?;
        if let Some(entry) = parse_ripgrep_event(text)? {
            matches.push(entry);
        }
    }
    Ok(matches)
}

fn parse_ripgrep_event(raw: &str) -> Result<Option<SearchMatchOutput>, String> {
    let event: RipgrepEvent =
        serde_json::from_str(raw).map_err(|err| format!("invalid ripgrep json event: {err}"))?;
    if event.kind != "match" {
        return Ok(None);
    }

    let data = event
        .data
        .ok_or_else(|| "invalid ripgrep match event: missing data".to_string())?;
    Ok(Some(SearchMatchOutput {
        path: decode_ripgrep_text(data.path, "path")?,
        line: data
            .line_number
            .ok_or_else(|| "invalid ripgrep match event: missing line_number".to_string())?,
        text: decode_ripgrep_text(data.lines, "lines")?
            .trim_end_matches('\n')
            .to_string(),
    }))
}

fn decode_ripgrep_text(field: Option<RipgrepTextField>, label: &str) -> Result<String, String> {
    let field =
        field.ok_or_else(|| format!("invalid ripgrep match event: missing {label} field"))?;
    if let Some(text) = field.text {
        return Ok(text);
    }

    let encoded = field
        .bytes
        .ok_or_else(|| format!("invalid ripgrep match event: missing {label} content"))?;
    let decoded = BASE64_STANDARD
        .decode(encoded)
        .map_err(|err| format!("invalid ripgrep {label} bytes: {err}"))?;
    String::from_utf8(decoded).map_err(|err| format!("invalid ripgrep {label} utf-8: {err}"))
}

fn sort_matches(matches: &mut [SearchMatchOutput]) {
    matches.sort_by(|left, right| {
        left.path
            .cmp(&right.path)
            .then(left.line.cmp(&right.line))
            .then(left.text.cmp(&right.text))
    });
}

#[derive(Debug, Deserialize)]
struct RipgrepEvent {
    #[serde(rename = "type")]
    kind: String,
    data: Option<RipgrepMatchData>,
}

#[derive(Debug, Deserialize)]
struct RipgrepMatchData {
    path: Option<RipgrepTextField>,
    lines: Option<RipgrepTextField>,
    line_number: Option<usize>,
}

#[derive(Debug, Deserialize)]
struct RipgrepTextField {
    text: Option<String>,
    bytes: Option<String>,
}
