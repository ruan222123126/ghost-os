use pyo3::prelude::*;
use serde_json::{json, to_string};

use crate::sandbox::tool_runtime::ToolRuntime;

use super::super::read_write::{apply_diff_impl, list_files_impl, read_file_impl, write_file_impl};

pub(crate) fn list_files_py(runtime: &ToolRuntime, path: Option<String>) -> PyResult<Vec<String>> {
    let target = normalize_path_arg(path.as_deref());
    let args = json!({ "path": &target });

    match list_files_impl(runtime.config(), &target) {
        Ok(result) => {
            runtime.log_success(
                "list_files",
                args,
                to_string(&result.entries).unwrap_or_default(),
            );
            Ok(result.entries)
        }
        Err(err) => Err(runtime.log_error("list_files", args, err)),
    }
}

pub(crate) fn read_file_py(
    runtime: &ToolRuntime,
    path: String,
    start_line: Option<usize>,
    end_line: Option<usize>,
) -> PyResult<String> {
    let args = json!({
        "path": &path,
        "start_line": start_line,
        "end_line": end_line,
    });

    match read_file_impl(runtime.config(), &path, start_line, end_line) {
        Ok(result) => {
            runtime.log_success("read_file", args, result.content.clone());
            Ok(result.content)
        }
        Err(err) => Err(runtime.log_error("read_file", args, err)),
    }
}

pub(crate) fn write_file_py(
    runtime: &ToolRuntime,
    path: String,
    content: String,
    mode: &str,
) -> PyResult<String> {
    let bytes = content.len();
    let lines = count_content_lines(&content);
    let args = json!({
        "path": &path,
        "mode": mode,
        "content_summary": {
            "bytes": bytes,
            "lines": lines,
        }
    });

    match write_file_impl(runtime.config(), &path, &content, mode) {
        Ok(result) => {
            runtime.log_success("write_file", args, result.message.clone());
            Ok(result.message)
        }
        Err(err) => Err(runtime.log_error("write_file", args, err)),
    }
}

pub(crate) fn apply_diff_py(
    runtime: &ToolRuntime,
    path: String,
    diff_text: String,
) -> PyResult<String> {
    match apply_diff_impl(runtime.config(), &path, &diff_text) {
        Ok(result) => {
            let hunk_ranges: Vec<serde_json::Value> = result
                .hunk_ranges
                .iter()
                .map(|hunk| {
                    json!({
                        "old_start": hunk.old_start,
                        "old_count": hunk.old_count,
                        "new_start": hunk.new_start,
                        "new_count": hunk.new_count,
                    })
                })
                .collect();
            let args = json!({
                "path": &path,
                "diff_summary": {
                    "hunk_count": result.hunk_count,
                    "added_lines": result.added_lines,
                    "removed_lines": result.removed_lines,
                    "hunk_ranges": hunk_ranges,
                }
            });
            runtime.log_success("apply_diff", args, result.message.clone());
            Ok(result.message)
        }
        Err(err) => {
            let args = json!({ "path": &path });
            Err(runtime.log_error("apply_diff", args, err))
        }
    }
}

fn normalize_path_arg(path: Option<&str>) -> String {
    path.map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".")
        .to_string()
}

fn count_content_lines(content: &str) -> usize {
    if content.is_empty() {
        0
    } else {
        content.lines().count()
    }
}
