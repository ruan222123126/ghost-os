use ignore::WalkBuilder;
use pyo3::prelude::*;
use regex::RegexBuilder;
use serde_json::{json, to_string};
use sha2::{Digest, Sha256};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::path::PathBuf;

use super::SandboxConfig;
use super::diff_engine::{PatchHunkRange, apply_unified_patch};
use super::path_policy::{is_blocked_path, resolve_read_path, resolve_write_path};
use super::tool_runtime::ToolRuntime;

pub(crate) struct ReadFileOutput {
    pub(crate) path: PathBuf,
    pub(crate) requested_start_line: usize,
    pub(crate) requested_end_line: usize,
    pub(crate) returned_start_line: usize,
    pub(crate) returned_end_line: usize,
    pub(crate) total_lines: usize,
    pub(crate) content: String,
}

pub(crate) struct ListFilesOutput {
    pub(crate) entries: Vec<String>,
}

pub(crate) struct WriteFileOutput {
    pub(crate) message: String,
}

pub(crate) struct ApplyDiffOutput {
    pub(crate) path: PathBuf,
    pub(crate) hunk_count: usize,
    pub(crate) added_lines: usize,
    pub(crate) removed_lines: usize,
    pub(crate) hunk_ranges: Vec<PatchHunkRange>,
    pub(crate) message: String,
}

pub(crate) struct SearchFilesOutput {
    pub(crate) dir_path: PathBuf,
    pub(crate) matches: Vec<String>,
}

pub(crate) struct ExportFileOutput {
    pub(crate) artifact_id: String,
    pub(crate) original_path: PathBuf,
    pub(crate) stored_path: PathBuf,
    pub(crate) filename: String,
    pub(crate) mime_type: String,
    pub(crate) bytes: usize,
    pub(crate) sha256: String,
}

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
    let bytes = content.as_bytes().len();
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

pub(crate) fn search_files_py(
    runtime: &ToolRuntime,
    keyword: String,
    dir_path: String,
    case_sensitive: bool,
) -> PyResult<Vec<String>> {
    let keyword = keyword.trim().to_string();
    let dir_path = normalize_path_arg(Some(&dir_path));
    let args = json!({
        "keyword": &keyword,
        "dir_path": &dir_path,
        "case_sensitive": case_sensitive,
    });

    match search_files_impl(runtime.config(), &keyword, &dir_path, case_sensitive) {
        Ok(result) => {
            runtime.log_success(
                "search_files",
                args,
                to_string(&result.matches).unwrap_or_default(),
            );
            Ok(result.matches)
        }
        Err(err) => Err(runtime.log_error("search_files", args, err)),
    }
}

pub(crate) fn read_file_impl(
    config: &SandboxConfig,
    path: &str,
    start_line: Option<usize>,
    end_line: Option<usize>,
) -> Result<ReadFileOutput, String> {
    let canonical = resolve_read_path(path, config)?;
    let content = fs::read_to_string(&canonical)
        .map_err(|err| format!("failed to read file {:?}: {err}", canonical))?;

    let requested_start_line = start_line.unwrap_or(1);
    if requested_start_line == 0 {
        return Err("start_line must be >= 1".to_string());
    }

    let requested_end_line = end_line.unwrap_or_else(|| {
        requested_start_line
            .saturating_add(config.max_file_read_lines)
            .saturating_sub(1)
    });
    if requested_end_line < requested_start_line {
        return Err("end_line must be >= start_line".to_string());
    }

    let requested = requested_end_line
        .saturating_sub(requested_start_line)
        .saturating_add(1);
    if requested > config.max_file_read_lines {
        return Err(format!(
            "read range too large (max {} lines)",
            config.max_file_read_lines
        ));
    }

    let lines: Vec<&str> = content.lines().collect();
    let total_lines = lines.len();
    let start_idx = requested_start_line.saturating_sub(1).min(total_lines);
    let end_idx = requested_end_line.min(total_lines);

    let content = if start_idx >= end_idx {
        String::new()
    } else {
        lines[start_idx..end_idx].join("\n")
    };
    let returned_line_count = end_idx.saturating_sub(start_idx);
    let returned_start_line = if returned_line_count == 0 {
        0
    } else {
        start_idx + 1
    };
    let returned_end_line = if returned_line_count == 0 {
        0
    } else {
        returned_start_line + returned_line_count - 1
    };

    Ok(ReadFileOutput {
        path: canonical,
        requested_start_line,
        requested_end_line,
        returned_start_line,
        returned_end_line,
        total_lines,
        content,
    })
}

pub(crate) fn export_file_impl(
    config: &SandboxConfig,
    path: &str,
    session_id: &str,
    artifact_root: &str,
    artifact_id: &str,
    max_bytes: Option<usize>,
) -> Result<ExportFileOutput, String> {
    let original_path = resolve_read_path(path, config)?;
    let metadata = fs::metadata(&original_path)
        .map_err(|err| format!("failed to stat file {:?}: {err}", original_path))?;
    if !metadata.is_file() {
        return Err(format!("path is not a file: {}", original_path.display()));
    }

    let file_size = usize::try_from(metadata.len()).map_err(|_| {
        format!(
            "file is too large to export on this platform: {}",
            original_path.display()
        )
    })?;
    if let Some(limit) = max_bytes {
        if file_size > limit {
            return Err(format!("file exceeds max_bytes limit ({limit} bytes)"));
        }
    }

    let stored_path = build_export_destination(
        config,
        artifact_root,
        session_id,
        artifact_id,
        &original_path,
    )?;
    let parent = stored_path
        .parent()
        .ok_or_else(|| "invalid export destination".to_string())?;
    fs::create_dir_all(parent)
        .map_err(|err| format!("failed to create artifact directory {:?}: {err}", parent))?;
    fs::copy(&original_path, &stored_path).map_err(|err| {
        format!(
            "failed to copy file {:?} to {:?}: {err}",
            original_path, stored_path
        )
    })?;

    let mut file = File::open(&stored_path)
        .map_err(|err| format!("failed to open exported file {:?}: {err}", stored_path))?;
    let mut hasher = Sha256::new();
    let mut buffer = [0u8; 8192];
    loop {
        let read = file
            .read(&mut buffer)
            .map_err(|err| format!("failed to read exported file {:?}: {err}", stored_path))?;
        if read == 0 {
            break;
        }
        hasher.update(&buffer[..read]);
    }

    let filename = original_path
        .file_name()
        .and_then(|value| value.to_str())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("failed to derive filename from {}", original_path.display()))?;

    Ok(ExportFileOutput {
        artifact_id: artifact_id.trim().to_string(),
        original_path,
        stored_path,
        filename: filename.clone(),
        mime_type: guess_mime_type(&filename),
        bytes: file_size,
        sha256: format!("{:x}", hasher.finalize()),
    })
}

pub(crate) fn list_files_impl(
    config: &SandboxConfig,
    path: &str,
) -> Result<ListFilesOutput, String> {
    let canonical = resolve_read_path(path, config)?;
    let entries = fs::read_dir(&canonical)
        .map_err(|err| format!("failed to read directory {:?}: {err}", canonical))?;

    let mut names = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|err| format!("failed to read directory entry: {err}"))?;
        let file_type = entry
            .file_type()
            .map_err(|err| format!("failed to read entry type: {err}"))?;

        let mut name = entry.file_name().to_string_lossy().to_string();
        if file_type.is_dir() {
            name.push('/');
        }
        names.push(name);
    }

    names.sort();

    Ok(ListFilesOutput { entries: names })
}

pub(crate) fn write_file_impl(
    config: &SandboxConfig,
    path: &str,
    content: &str,
    mode: &str,
) -> Result<WriteFileOutput, String> {
    let bytes = content.as_bytes();
    if bytes.len() > config.max_file_write_bytes {
        return Err(format!(
            "content too large (max {} bytes)",
            config.max_file_write_bytes
        ));
    }

    let canonical = resolve_write_path(path, config)?;
    let mode = mode.trim().to_ascii_lowercase();
    if mode != "write" && mode != "append" {
        return Err("mode must be 'write' or 'append'".to_string());
    }

    let parent = canonical
        .parent()
        .ok_or_else(|| "invalid write path".to_string())?;
    if !parent.exists() {
        return Err(format!(
            "parent directory does not exist: {}",
            parent.display()
        ));
    }

    let mut options = OpenOptions::new();
    options.write(true).create(true);
    if mode == "append" {
        options.append(true);
    } else {
        options.truncate(true);
    }

    let mut file = options
        .open(&canonical)
        .map_err(|err| format!("failed to open file {:?}: {err}", canonical))?;
    file.write_all(bytes)
        .map_err(|err| format!("failed to write file {:?}: {err}", canonical))?;

    Ok(WriteFileOutput {
        message: format!("wrote {} bytes to {}", bytes.len(), canonical.display()),
    })
}

pub(crate) fn apply_diff_impl(
    config: &SandboxConfig,
    path: &str,
    diff_text: &str,
) -> Result<ApplyDiffOutput, String> {
    if diff_text.trim().is_empty() {
        return Err("diff_text is required".to_string());
    }

    let canonical = resolve_write_path(path, config)?;
    let original = fs::read_to_string(&canonical)
        .map_err(|err| format!("failed to read file {:?}: {err}", canonical))?;
    let patch = apply_unified_patch(&original, diff_text)?;

    if patch.updated.len() > config.max_file_write_bytes {
        return Err(format!(
            "patched file exceeds max write size ({} bytes)",
            config.max_file_write_bytes
        ));
    }

    fs::write(&canonical, patch.updated.as_bytes())
        .map_err(|err| format!("failed to write file {:?}: {err}", canonical))?;

    Ok(ApplyDiffOutput {
        path: canonical.clone(),
        hunk_count: patch.hunk_count,
        added_lines: patch.added_lines,
        removed_lines: patch.removed_lines,
        hunk_ranges: patch.hunk_ranges,
        message: format!(
            "applied {} hunks to {}",
            patch.hunk_count,
            canonical.display()
        ),
    })
}

pub(crate) fn search_files_impl(
    config: &SandboxConfig,
    keyword: &str,
    dir_path: &str,
    case_sensitive: bool,
) -> Result<SearchFilesOutput, String> {
    let keyword = keyword.trim();
    if keyword.is_empty() {
        return Err("keyword is required".to_string());
    }

    let canonical_dir = resolve_read_path(dir_path, config)?;
    if !canonical_dir.is_dir() {
        return Err(format!(
            "search path is not a directory: {}",
            canonical_dir.display()
        ));
    }

    let matcher = RegexBuilder::new(keyword)
        .case_insensitive(!case_sensitive)
        .build()
        .map_err(|err| format!("invalid regex pattern: {err}"))?;

    let mut walk_builder = WalkBuilder::new(&canonical_dir);
    walk_builder.standard_filters(true);
    walk_builder.max_depth(Some(config.max_search_depth));

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
        if files_scanned > config.max_search_files {
            break;
        }

        let path = entry.path();
        if is_blocked_path(path, &config.blocked_patterns) {
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
                if matches.len() >= config.max_search_matches {
                    break 'walk;
                }
            }
        }
    }

    Ok(SearchFilesOutput {
        dir_path: canonical_dir,
        matches,
    })
}

fn build_export_destination(
    config: &SandboxConfig,
    artifact_root: &str,
    session_id: &str,
    artifact_id: &str,
    original_path: &PathBuf,
) -> Result<PathBuf, String> {
    let session_id = sanitize_identifier(session_id, "session_id")?;
    let artifact_id = sanitize_identifier(artifact_id, "artifact_id")?;
    let extension = original_path
        .extension()
        .and_then(|value| value.to_str())
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(|value| format!(".{value}"))
        .unwrap_or_default();
    let destination = PathBuf::from(artifact_root)
        .join("sessions")
        .join(session_id)
        .join(format!("{artifact_id}{extension}"));
    resolve_write_path(&destination.to_string_lossy(), config)
}

fn sanitize_identifier(value: &str, label: &str) -> Result<String, String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return Err(format!("{label} is required"));
    }
    if trimmed.contains('/') || trimmed.contains('\\') || trimmed == "." || trimmed == ".." {
        return Err(format!("invalid {label}"));
    }
    Ok(trimmed.to_string())
}

fn guess_mime_type(filename: &str) -> String {
    match filename
        .rsplit('.')
        .next()
        .map(|value| value.to_ascii_lowercase())
    {
        Some(ext) if ext == "txt" => "text/plain".to_string(),
        Some(ext) if ext == "md" => "text/markdown".to_string(),
        Some(ext) if ext == "json" => "application/json".to_string(),
        Some(ext) if ext == "pdf" => "application/pdf".to_string(),
        Some(ext) if ext == "png" => "image/png".to_string(),
        Some(ext) if ext == "jpg" || ext == "jpeg" => "image/jpeg".to_string(),
        Some(ext) if ext == "gif" => "image/gif".to_string(),
        Some(ext) if ext == "webp" => "image/webp".to_string(),
        Some(ext) if ext == "csv" => "text/csv".to_string(),
        Some(ext) if ext == "html" || ext == "htm" => "text/html".to_string(),
        Some(ext) if ext == "xml" => "application/xml".to_string(),
        Some(ext) if ext == "zip" => "application/zip".to_string(),
        Some(ext) if ext == "tar" => "application/x-tar".to_string(),
        Some(ext) if ext == "gz" => "application/gzip".to_string(),
        Some(ext) if ext == "mp4" => "video/mp4".to_string(),
        Some(ext) if ext == "mp3" => "audio/mpeg".to_string(),
        Some(ext) if ext == "wav" => "audio/wav".to_string(),
        _ => "application/octet-stream".to_string(),
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
