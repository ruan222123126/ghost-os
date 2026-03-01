// Native sandbox tool adapters that translate bridge tool calls into atomic operations.

use ignore::WalkBuilder;
use pyo3::exceptions::PyRuntimeError;
use pyo3::prelude::*;
use regex::RegexBuilder;
use serde_json::{json, to_string};
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::process::{Command, Output};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};

use super::diff_engine::apply_unified_patch;
use super::path_policy::{is_blocked_path, resolve_read_path, resolve_write_path};
use super::web_security::{fetch_and_convert_webpage, validate_https_url};
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

        let patch = match apply_unified_patch(&original, &diff_text) {
            Ok(patch) => patch,
            Err(err) => return Err(self.log_tool_error("apply_diff", args, err)),
        };

        if patch.updated.as_bytes().len() > self.config.max_file_write_bytes {
            return Err(self.log_tool_error(
                "apply_diff",
                args,
                format!(
                    "patched file exceeds max write size ({} bytes)",
                    self.config.max_file_write_bytes
                ),
            ));
        }

        if let Err(err) = fs::write(&canonical, patch.updated.as_bytes()) {
            return Err(self.log_tool_error(
                "apply_diff",
                args,
                format!("failed to write file {:?}: {err}", canonical),
            ));
        }

        let result = format!(
            "applied {} hunks to {}",
            patch.hunk_count,
            canonical.display()
        );
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

        let parsed = match validate_https_url(&url) {
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

fn push_tool_log(tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>, log: ToolCallLog) {
    if let Ok(mut logs) = tool_calls_log.lock() {
        logs.push(log);
    }
}
