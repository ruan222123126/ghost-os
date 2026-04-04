use std::fs;
#[cfg(feature = "python-sandbox")]
use std::fs::OpenOptions;
#[cfg(feature = "python-sandbox")]
use std::io::Write;
use std::path::PathBuf;

use crate::sandbox::SandboxConfig;
use crate::sandbox::diff_engine::{PatchHunkRange, apply_unified_patch};
use crate::sandbox::path_policy::{resolve_read_path, resolve_write_path};

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

#[cfg(feature = "python-sandbox")]
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
    let (returned_start_line, returned_end_line, content) =
        slice_requested_lines(&lines, start_idx, end_idx);

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

#[cfg(feature = "python-sandbox")]
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

fn slice_requested_lines(
    lines: &[&str],
    start_idx: usize,
    end_idx: usize,
) -> (usize, usize, String) {
    if start_idx >= end_idx {
        return (0, 0, String::new());
    }

    let content = lines[start_idx..end_idx].join("\n");
    let returned_start_line = start_idx + 1;
    let returned_end_line = returned_start_line + end_idx.saturating_sub(start_idx) - 1;
    (returned_start_line, returned_end_line, content)
}
