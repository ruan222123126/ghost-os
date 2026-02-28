use glob::Pattern;
use ignore::WalkBuilder;
use pyo3::exceptions::PyRuntimeError;
use pyo3::prelude::*;
use regex::{Regex, RegexBuilder};
use reqwest::Url;
use reqwest::blocking::Client;
use serde_json::{json, to_string};
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::net::{IpAddr, ToSocketAddrs};
use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex, OnceLock};
use std::time::Duration;

use super::{SandboxConfig, ToolCallLog};

const MAX_BASH_OUTPUT_CHARS: usize = 2_000;

#[pyclass]
pub struct ToolsProxy {
    tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>,
    config: SandboxConfig,
    web_request_count: Arc<AtomicUsize>,
}

impl ToolsProxy {
    pub fn new(tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>, config: SandboxConfig) -> Self {
        Self {
            tool_calls_log,
            config,
            web_request_count: Arc::new(AtomicUsize::new(0)),
        }
    }
}

#[allow(unsafe_op_in_unsafe_fn)]
#[pymethods]
impl ToolsProxy {
    #[pyo3(signature = (*, command))]
    fn bash_exec(&self, py: Python<'_>, command: String) -> PyResult<String> {
        let command = command.trim().to_string();
        if command.is_empty() {
            return Err(self.log_tool_error(
                "bash_exec",
                json!({ "command": command }),
                "command is required".to_string(),
            ));
        }

        let output = py.allow_threads(|| run_shell_command(&command));
        match output {
            Ok(output) => {
                let stdout = truncate_output(&String::from_utf8_lossy(&output.stdout));
                let stderr = String::from_utf8_lossy(&output.stderr).to_string();

                if !output.status.success() {
                    let err_msg = if stderr.trim().is_empty() {
                        format!("command failed: exit status {}", output.status)
                    } else {
                        format!("command failed: {}", stderr.trim())
                    };
                    push_tool_log(
                        &self.tool_calls_log,
                        ToolCallLog {
                            tool: "bash_exec".to_string(),
                            args: json!({ "command": command }),
                            result: stdout,
                            error: Some(err_msg.clone()),
                        },
                    );
                    return Err(PyRuntimeError::new_err(err_msg));
                }

                push_tool_log(
                    &self.tool_calls_log,
                    ToolCallLog {
                        tool: "bash_exec".to_string(),
                        args: json!({ "command": command }),
                        result: stdout.clone(),
                        error: None,
                    },
                );
                Ok(stdout)
            }
            Err(err) => {
                let err_msg = format!("failed to execute command: {err}");
                Err(self.log_tool_error("bash_exec", json!({ "command": command }), err_msg))
            }
        }
    }

    #[pyo3(signature = (*, path=None))]
    fn list_files(&self, path: Option<String>) -> PyResult<Vec<String>> {
        let input = path.unwrap_or_else(|| ".".to_string());
        let trimmed = input.trim();
        let target = if trimmed.is_empty() { "." } else { trimmed };

        let canonical = match resolve_read_path(target, &self.config) {
            Ok(path) => path,
            Err(err) => {
                return Err(self.log_tool_error("list_files", json!({ "path": target }), err));
            }
        };

        match fs::read_dir(&canonical) {
            Ok(entries) => {
                let mut files = Vec::new();
                for entry in entries {
                    let entry = match entry {
                        Ok(entry) => entry,
                        Err(err) => {
                            return Err(self.log_tool_error(
                                "list_files",
                                json!({ "path": target }),
                                format!("failed to read directory entry: {err}"),
                            ));
                        }
                    };

                    let file_type = match entry.file_type() {
                        Ok(file_type) => file_type,
                        Err(err) => {
                            return Err(self.log_tool_error(
                                "list_files",
                                json!({ "path": target }),
                                format!("failed to read entry type: {err}"),
                            ));
                        }
                    };

                    let mut name = entry.file_name().to_string_lossy().to_string();
                    if file_type.is_dir() {
                        name.push('/');
                    }
                    files.push(name);
                }

                files.sort();
                push_tool_log(
                    &self.tool_calls_log,
                    ToolCallLog {
                        tool: "list_files".to_string(),
                        args: json!({ "path": target }),
                        result: to_string(&files).unwrap_or_default(),
                        error: None,
                    },
                );
                Ok(files)
            }
            Err(err) => Err(self.log_tool_error(
                "list_files",
                json!({ "path": target }),
                format!("failed to read directory {target:?}: {err}"),
            )),
        }
    }

    #[pyo3(signature = (*, path, start_line=None, end_line=None))]
    fn read_file(
        &self,
        path: String,
        start_line: Option<usize>,
        end_line: Option<usize>,
    ) -> PyResult<String> {
        let args = json!({
            "path": &path,
            "start_line": start_line,
            "end_line": end_line,
        });

        let canonical = match resolve_read_path(&path, &self.config) {
            Ok(path) => path,
            Err(err) => return Err(self.log_tool_error("read_file", args, err)),
        };

        let content = match fs::read_to_string(&canonical) {
            Ok(content) => content,
            Err(err) => {
                return Err(self.log_tool_error(
                    "read_file",
                    args,
                    format!("failed to read file {:?}: {err}", canonical),
                ));
            }
        };

        let start = start_line.unwrap_or(1);
        if start == 0 {
            return Err(self.log_tool_error(
                "read_file",
                args,
                "start_line must be >= 1".to_string(),
            ));
        }

        let end = end_line.unwrap_or_else(|| {
            start
                .saturating_add(self.config.max_file_read_lines)
                .saturating_sub(1)
        });
        if end < start {
            return Err(self.log_tool_error(
                "read_file",
                args,
                "end_line must be >= start_line".to_string(),
            ));
        }

        let requested = end.saturating_sub(start).saturating_add(1);
        if requested > self.config.max_file_read_lines {
            return Err(self.log_tool_error(
                "read_file",
                args,
                format!(
                    "read range too large (max {} lines)",
                    self.config.max_file_read_lines
                ),
            ));
        }

        let lines: Vec<&str> = content.lines().collect();
        let start_idx = start.saturating_sub(1).min(lines.len());
        let end_idx = end.min(lines.len());

        let result = if start_idx >= end_idx {
            String::new()
        } else {
            lines[start_idx..end_idx].join("\n")
        };

        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: "read_file".to_string(),
                args,
                result: truncate_output(&result),
                error: None,
            },
        );
        Ok(result)
    }

    #[pyo3(signature = (*, path, content, mode="write"))]
    fn write_file(&self, path: String, content: String, mode: &str) -> PyResult<String> {
        let args = json!({
            "path": &path,
            "mode": mode,
        });

        let bytes = content.as_bytes();
        if bytes.len() > self.config.max_file_write_bytes {
            return Err(self.log_tool_error(
                "write_file",
                args,
                format!(
                    "content too large (max {} bytes)",
                    self.config.max_file_write_bytes
                ),
            ));
        }

        let canonical = match resolve_write_path(&path, &self.config) {
            Ok(path) => path,
            Err(err) => return Err(self.log_tool_error("write_file", args, err)),
        };

        let mode = mode.trim().to_ascii_lowercase();
        if mode != "write" && mode != "append" {
            return Err(self.log_tool_error(
                "write_file",
                args,
                "mode must be 'write' or 'append'".to_string(),
            ));
        }

        let parent = match canonical.parent() {
            Some(parent) => parent,
            None => {
                return Err(self.log_tool_error(
                    "write_file",
                    args,
                    "invalid write path".to_string(),
                ));
            }
        };

        if !parent.exists() {
            return Err(self.log_tool_error(
                "write_file",
                args,
                format!("parent directory does not exist: {}", parent.display()),
            ));
        }

        let mut options = OpenOptions::new();
        options.write(true).create(true);
        if mode == "append" {
            options.append(true);
        } else {
            options.truncate(true);
        }

        let mut file = match options.open(&canonical) {
            Ok(file) => file,
            Err(err) => {
                return Err(self.log_tool_error(
                    "write_file",
                    args,
                    format!("failed to open file {:?}: {err}", canonical),
                ));
            }
        };

        if let Err(err) = file.write_all(bytes) {
            return Err(self.log_tool_error(
                "write_file",
                args,
                format!("failed to write file {:?}: {err}", canonical),
            ));
        }

        let result = format!("wrote {} bytes to {}", bytes.len(), canonical.display());
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: "write_file".to_string(),
                args,
                result: result.clone(),
                error: None,
            },
        );

        Ok(result)
    }

    #[pyo3(signature = (*, path, diff_text))]
    fn apply_diff(&self, path: String, diff_text: String) -> PyResult<String> {
        let args = json!({ "path": &path });

        if diff_text.trim().is_empty() {
            return Err(self.log_tool_error(
                "apply_diff",
                args,
                "diff_text is required".to_string(),
            ));
        }

        let canonical = match resolve_write_path(&path, &self.config) {
            Ok(path) => path,
            Err(err) => return Err(self.log_tool_error("apply_diff", args, err)),
        };

        let original = match fs::read_to_string(&canonical) {
            Ok(content) => content,
            Err(err) => {
                return Err(self.log_tool_error(
                    "apply_diff",
                    args,
                    format!("failed to read file {:?}: {err}", canonical),
                ));
            }
        };

        let hunks = match parse_unified_diff(&diff_text) {
            Ok(hunks) => hunks,
            Err(err) => return Err(self.log_tool_error("apply_diff", args, err)),
        };

        let updated = match apply_unified_diff(&original, &hunks) {
            Ok(updated) => updated,
            Err(err) => return Err(self.log_tool_error("apply_diff", args, err)),
        };

        if updated.as_bytes().len() > self.config.max_file_write_bytes {
            return Err(self.log_tool_error(
                "apply_diff",
                args,
                format!(
                    "patched file exceeds max write size ({} bytes)",
                    self.config.max_file_write_bytes
                ),
            ));
        }

        if let Err(err) = fs::write(&canonical, updated.as_bytes()) {
            return Err(self.log_tool_error(
                "apply_diff",
                args,
                format!("failed to write file {:?}: {err}", canonical),
            ));
        }

        let result = format!("applied {} hunks to {}", hunks.len(), canonical.display());
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: "apply_diff".to_string(),
                args,
                result: result.clone(),
                error: None,
            },
        );
        Ok(result)
    }

    #[pyo3(signature = (*, keyword, dir_path=".".to_string(), case_sensitive=true))]
    fn search_files(
        &self,
        keyword: String,
        dir_path: String,
        case_sensitive: bool,
    ) -> PyResult<Vec<String>> {
        let keyword = keyword.trim();
        let args = json!({
            "keyword": keyword,
            "dir_path": &dir_path,
            "case_sensitive": case_sensitive,
        });

        if keyword.is_empty() {
            return Err(self.log_tool_error(
                "search_files",
                args,
                "keyword is required".to_string(),
            ));
        }

        let canonical_dir = match resolve_read_path(&dir_path, &self.config) {
            Ok(path) => path,
            Err(err) => return Err(self.log_tool_error("search_files", args, err)),
        };

        if !canonical_dir.is_dir() {
            return Err(self.log_tool_error(
                "search_files",
                args,
                format!(
                    "search path is not a directory: {}",
                    canonical_dir.display()
                ),
            ));
        }

        let matcher = match RegexBuilder::new(keyword)
            .case_insensitive(!case_sensitive)
            .build()
        {
            Ok(regex) => regex,
            Err(err) => {
                return Err(self.log_tool_error(
                    "search_files",
                    args,
                    format!("invalid regex pattern: {err}"),
                ));
            }
        };

        let mut walk_builder = WalkBuilder::new(&canonical_dir);
        walk_builder.standard_filters(true);
        walk_builder.max_depth(Some(self.config.max_search_depth));

        let mut files_scanned = 0usize;
        let mut matches = Vec::new();

        'walk: for entry in walk_builder.build() {
            let entry = match entry {
                Ok(entry) => entry,
                Err(_) => continue,
            };

            let file_type = match entry.file_type() {
                Some(file_type) => file_type,
                None => continue,
            };
            if !file_type.is_file() {
                continue;
            }

            files_scanned += 1;
            if files_scanned > self.config.max_search_files {
                break;
            }

            let path = entry.path();
            if is_blocked_path(path, &self.config.blocked_patterns) {
                continue;
            }

            let content = match fs::read(path) {
                Ok(content) => content,
                Err(_) => continue,
            };

            if content.contains(&0) {
                continue;
            }

            let text = String::from_utf8_lossy(&content);
            for (line_number, line) in text.lines().enumerate() {
                if matcher.is_match(line) {
                    let display_path = path
                        .strip_prefix(&canonical_dir)
                        .unwrap_or(path)
                        .to_string_lossy()
                        .to_string();
                    matches.push(format!("{}:{}:{}", display_path, line_number + 1, line));
                    if matches.len() >= self.config.max_search_matches {
                        break 'walk;
                    }
                }
            }
        }

        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: "search_files".to_string(),
                args,
                result: to_string(&matches).unwrap_or_default(),
                error: None,
            },
        );
        Ok(matches)
    }

    #[pyo3(signature = (*, url))]
    fn fetch_webpage(&self, py: Python<'_>, url: String) -> PyResult<String> {
        let args = json!({ "url": &url });

        let request_number = self.web_request_count.fetch_add(1, Ordering::SeqCst) + 1;
        if request_number > self.config.max_web_requests {
            return Err(self.log_tool_error(
                "fetch_webpage",
                args,
                format!(
                    "rate limit: max {} web requests per script",
                    self.config.max_web_requests
                ),
            ));
        }

        let parsed = match validate_url(&url) {
            Ok(parsed) => parsed,
            Err(err) => return Err(self.log_tool_error("fetch_webpage", args, err)),
        };

        let fetch_result =
            py.allow_threads(|| fetch_and_convert_webpage(&parsed, self.config.max_webpage_bytes));
        match fetch_result {
            Ok(markdown) => {
                let markdown = truncate_to_bytes(&markdown, self.config.max_webpage_bytes);
                push_tool_log(
                    &self.tool_calls_log,
                    ToolCallLog {
                        tool: "fetch_webpage".to_string(),
                        args,
                        result: markdown.clone(),
                        error: None,
                    },
                );
                Ok(markdown)
            }
            Err(err) => Err(self.log_tool_error("fetch_webpage", args, err)),
        }
    }
}

impl ToolsProxy {
    fn log_tool_error(&self, tool: &str, args: serde_json::Value, err: String) -> PyErr {
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: tool.to_string(),
                args,
                result: String::new(),
                error: Some(err.clone()),
            },
        );
        PyRuntimeError::new_err(err)
    }
}

fn run_shell_command(command: &str) -> std::io::Result<Output> {
    if cfg!(target_os = "windows") {
        Command::new("cmd").arg("/C").arg(command).output()
    } else {
        Command::new("bash").arg("-lc").arg(command).output()
    }
}

fn truncate_output(text: &str) -> String {
    let char_count = text.chars().count();
    if char_count <= MAX_BASH_OUTPUT_CHARS {
        return text.to_string();
    }

    let truncated: String = text.chars().take(MAX_BASH_OUTPUT_CHARS).collect();
    format!(
        "{}\n\n[... output truncated, {} more chars. Use grep/head or write to file for full output]",
        truncated,
        char_count - MAX_BASH_OUTPUT_CHARS
    )
}

fn truncate_to_bytes(text: &str, max_bytes: usize) -> String {
    if text.len() <= max_bytes {
        return text.to_string();
    }

    let mut end = max_bytes;
    while !text.is_char_boundary(end) {
        end = end.saturating_sub(1);
        if end == 0 {
            return String::new();
        }
    }
    text[..end].to_string()
}

fn resolve_read_path(path: &str, config: &SandboxConfig) -> Result<PathBuf, String> {
    let canonical = canonicalize_for_read(path)?;
    ensure_path_allowed(&canonical, &config.allowed_read_paths)?;
    ensure_not_blocked(&canonical, &config.blocked_patterns)?;
    Ok(canonical)
}

fn resolve_write_path(path: &str, config: &SandboxConfig) -> Result<PathBuf, String> {
    let canonical = canonicalize_for_write(path)?;
    ensure_path_allowed(&canonical, &config.allowed_write_paths)?;
    ensure_not_blocked(&canonical, &config.blocked_patterns)?;
    Ok(canonical)
}

fn canonicalize_for_read(path: &str) -> Result<PathBuf, String> {
    let absolute = to_absolute_path(path)?;
    if !absolute.exists() {
        return Err(format!("path does not exist: {}", absolute.display()));
    }
    fs::canonicalize(&absolute).map_err(|err| format!("invalid path {}: {err}", absolute.display()))
}

fn canonicalize_for_write(path: &str) -> Result<PathBuf, String> {
    let absolute = to_absolute_path(path)?;
    if absolute.exists() {
        return fs::canonicalize(&absolute)
            .map_err(|err| format!("invalid path {}: {err}", absolute.display()));
    }

    let (ancestor, suffix) = split_existing_ancestor(&absolute)?;
    let canonical_ancestor = fs::canonicalize(&ancestor)
        .map_err(|err| format!("invalid path {}: {err}", ancestor.display()))?;
    Ok(canonical_ancestor.join(suffix))
}

fn to_absolute_path(path: &str) -> Result<PathBuf, String> {
    let trimmed = path.trim();
    if trimmed.is_empty() {
        return Err("path is required".to_string());
    }

    let candidate = PathBuf::from(trimmed);
    if candidate.is_absolute() {
        Ok(candidate)
    } else {
        std::env::current_dir()
            .map(|cwd| cwd.join(candidate))
            .map_err(|err| format!("failed to resolve current directory: {err}"))
    }
}

fn split_existing_ancestor(path: &Path) -> Result<(PathBuf, PathBuf), String> {
    let mut ancestor = path.to_path_buf();
    let mut suffix = PathBuf::new();

    while !ancestor.exists() {
        let file_name = ancestor
            .file_name()
            .ok_or_else(|| format!("invalid path: {}", path.display()))?;
        let mut next_suffix = PathBuf::from(file_name);
        if !suffix.as_os_str().is_empty() {
            next_suffix.push(&suffix);
        }
        suffix = next_suffix;

        ancestor = ancestor
            .parent()
            .ok_or_else(|| format!("invalid path: {}", path.display()))?
            .to_path_buf();
    }

    Ok((ancestor, suffix))
}

fn ensure_path_allowed(path: &Path, allowed_dirs: &[String]) -> Result<(), String> {
    let mut canonical_allowed = Vec::new();
    for dir in allowed_dirs {
        let absolute = to_absolute_path(dir)?;
        let canonical = fs::canonicalize(&absolute)
            .map_err(|err| format!("invalid allowed directory {}: {err}", absolute.display()))?;
        canonical_allowed.push(canonical);
    }

    if canonical_allowed.is_empty() {
        return Err("no allowed directories configured".to_string());
    }

    let is_allowed = canonical_allowed
        .iter()
        .any(|allowed| path.starts_with(allowed));
    if !is_allowed {
        return Err("path not in allowed directories".to_string());
    }

    Ok(())
}

fn ensure_not_blocked(path: &Path, blocked_patterns: &[String]) -> Result<(), String> {
    if is_blocked_path(path, blocked_patterns) {
        return Err("access to sensitive file blocked".to_string());
    }
    Ok(())
}

fn is_blocked_path(path: &Path, blocked_patterns: &[String]) -> bool {
    let file_name = path
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("");
    blocked_patterns.iter().any(|pattern| {
        Pattern::new(pattern)
            .map(|glob| glob.matches(file_name) || glob.matches_path(path))
            .unwrap_or(false)
    })
}

#[derive(Debug, Clone)]
struct DiffHunk {
    old_start: usize,
    lines: Vec<DiffLine>,
}

#[derive(Debug, Clone)]
enum DiffLine {
    Context(String),
    Add(String),
    Remove(String),
}

fn hunk_header_regex() -> &'static Regex {
    static HUNK_RE: OnceLock<Regex> = OnceLock::new();
    HUNK_RE.get_or_init(|| {
        Regex::new(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@").expect("valid hunk regex")
    })
}

fn parse_unified_diff(diff_text: &str) -> Result<Vec<DiffHunk>, String> {
    let mut lines = diff_text.lines().peekable();
    let mut hunks = Vec::new();
    let hunk_re = hunk_header_regex();

    while let Some(raw_line) = lines.next() {
        let line = raw_line.strip_suffix('\r').unwrap_or(raw_line);
        let captures = match hunk_re.captures(line) {
            Some(captures) => captures,
            None => continue,
        };

        let old_start = captures
            .get(1)
            .and_then(|value| value.as_str().parse::<usize>().ok())
            .ok_or_else(|| format!("invalid hunk header: {line}"))?;

        let mut hunk_lines = Vec::new();
        while let Some(next) = lines.peek() {
            let candidate = next.strip_suffix('\r').unwrap_or(next);
            if hunk_re.is_match(candidate) {
                break;
            }

            let next = lines.next().unwrap_or_default();
            let candidate = next.strip_suffix('\r').unwrap_or(next);
            if candidate == "\\ No newline at end of file" {
                continue;
            }

            let mut chars = candidate.chars();
            let marker = chars
                .next()
                .ok_or_else(|| "malformed diff line".to_string())?;
            let value: String = chars.collect();
            match marker {
                ' ' => hunk_lines.push(DiffLine::Context(value)),
                '+' => hunk_lines.push(DiffLine::Add(value)),
                '-' => hunk_lines.push(DiffLine::Remove(value)),
                _ => return Err(format!("malformed diff line: {candidate}")),
            }
        }

        if hunk_lines.is_empty() {
            return Err("diff hunk has no body".to_string());
        }

        hunks.push(DiffHunk {
            old_start,
            lines: hunk_lines,
        });
    }

    if hunks.is_empty() {
        return Err("no diff hunks found".to_string());
    }

    Ok(hunks)
}

fn apply_unified_diff(original: &str, hunks: &[DiffHunk]) -> Result<String, String> {
    let source_lines: Vec<&str> = original.lines().collect();
    let had_trailing_newline = original.ends_with('\n');

    let mut result = Vec::new();
    let mut source_index = 0usize;

    for hunk in hunks {
        let hunk_start = hunk.old_start.saturating_sub(1);
        if hunk_start < source_index {
            return Err("invalid diff order: overlapping hunks".to_string());
        }
        if hunk_start > source_lines.len() {
            return Err(format!(
                "hunk start {} exceeds file length {}",
                hunk.old_start,
                source_lines.len()
            ));
        }

        for line in &source_lines[source_index..hunk_start] {
            result.push((*line).to_string());
        }

        let mut cursor = hunk_start;
        for line in &hunk.lines {
            match line {
                DiffLine::Context(expected) => {
                    let actual = source_lines
                        .get(cursor)
                        .ok_or_else(|| "diff context exceeds file length".to_string())?;
                    if actual != &expected.as_str() {
                        return Err(format!(
                            "diff context mismatch at line {}",
                            cursor.saturating_add(1)
                        ));
                    }
                    result.push(expected.clone());
                    cursor = cursor.saturating_add(1);
                }
                DiffLine::Remove(expected) => {
                    let actual = source_lines
                        .get(cursor)
                        .ok_or_else(|| "diff removal exceeds file length".to_string())?;
                    if actual != &expected.as_str() {
                        return Err(format!(
                            "diff removal mismatch at line {}",
                            cursor.saturating_add(1)
                        ));
                    }
                    cursor = cursor.saturating_add(1);
                }
                DiffLine::Add(value) => {
                    result.push(value.clone());
                }
            }
        }

        source_index = cursor;
    }

    for line in &source_lines[source_index..] {
        result.push((*line).to_string());
    }

    let mut patched = result.join("\n");
    if had_trailing_newline {
        patched.push('\n');
    }
    Ok(patched)
}

fn validate_url(raw_url: &str) -> Result<Url, String> {
    let trimmed = raw_url.trim();
    if trimmed.is_empty() {
        return Err("url is required".to_string());
    }

    let parsed = Url::parse(trimmed).map_err(|err| format!("invalid URL: {err}"))?;
    if parsed.scheme() != "https" {
        return Err("only HTTPS URLs allowed".to_string());
    }

    let host = parsed
        .host_str()
        .ok_or_else(|| "URL must include host".to_string())?;
    if host.eq_ignore_ascii_case("localhost") || host.ends_with(".localhost") {
        return Err("private IP addresses blocked".to_string());
    }

    if let Ok(ip) = host.parse::<IpAddr>() {
        if is_private_ip(ip) {
            return Err("private IP addresses blocked".to_string());
        }
    } else {
        let port = parsed.port_or_known_default().unwrap_or(443);
        let mut resolved_any = false;
        for addr in (host, port)
            .to_socket_addrs()
            .map_err(|err| format!("host resolution failed: {err}"))?
        {
            resolved_any = true;
            if is_private_ip(addr.ip()) {
                return Err("private IP addresses blocked".to_string());
            }
        }

        if !resolved_any {
            return Err("host resolution failed".to_string());
        }
    }

    Ok(parsed)
}

fn is_private_ip(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(ipv4) => {
            let octets = ipv4.octets();
            let in_cgnat = octets[0] == 100 && (octets[1] & 0b1100_0000) == 0b0100_0000;
            ipv4.is_private()
                || ipv4.is_loopback()
                || ipv4.is_link_local()
                || ipv4.is_multicast()
                || ipv4.is_broadcast()
                || ipv4.is_documentation()
                || in_cgnat
                || octets[0] == 0
        }
        IpAddr::V6(ipv6) => {
            ipv6.is_loopback()
                || ipv6.is_unspecified()
                || ipv6.is_unique_local()
                || ipv6.is_multicast()
                || ipv6.is_unicast_link_local()
        }
    }
}

fn fetch_and_convert_webpage(url: &Url, max_bytes: usize) -> Result<String, String> {
    let client = Client::builder()
        .timeout(Duration::from_secs(10))
        .user_agent("Ghost-OS/1.0")
        .redirect(reqwest::redirect::Policy::limited(5))
        .build()
        .map_err(|err| format!("failed to initialize HTTP client: {err}"))?;

    let response = client
        .get(url.clone())
        .send()
        .map_err(|err| format!("request failed: {err}"))?;

    if !response.status().is_success() {
        return Err(format!("request failed with status {}", response.status()));
    }

    if let Some(content_length) = response.content_length() {
        if content_length > max_bytes as u64 {
            return Err(format!("response too large (max {} bytes)", max_bytes));
        }
    }

    let mut body = Vec::new();
    response
        .take(max_bytes as u64 + 1)
        .read_to_end(&mut body)
        .map_err(|err| format!("failed to read response body: {err}"))?;

    if body.len() > max_bytes {
        return Err(format!("response too large (max {} bytes)", max_bytes));
    }

    Ok(html2text::from_read(body.as_slice(), 100))
}

fn push_tool_log(tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>, log: ToolCallLog) {
    if let Ok(mut logs) = tool_calls_log.lock() {
        logs.push(log);
    }
}
