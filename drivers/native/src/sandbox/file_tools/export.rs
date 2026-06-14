use sha2::{Digest, Sha256};
use std::fs::{self, File};
use std::io::Read;
use std::path::{Path, PathBuf};

use crate::sandbox::SandboxConfig;
use crate::sandbox::path_policy::{resolve_read_path, resolve_write_path};

pub(crate) struct ExportFileOutput {
    pub(crate) artifact_id: String,
    pub(crate) original_path: PathBuf,
    pub(crate) stored_path: PathBuf,
    pub(crate) filename: String,
    pub(crate) mime_type: String,
    pub(crate) bytes: usize,
    pub(crate) sha256: String,
}

pub(crate) fn export_file_impl(
    config: &SandboxConfig,
    path: &str,
    session_id: &str,
    artifact_root: &str,
    artifact_id: &str,
    max_bytes: Option<usize>,
) -> Result<ExportFileOutput, String> {
    let source = resolve_export_source(config, path, max_bytes)?;
    let stored_path = build_export_destination(
        config,
        artifact_root,
        session_id,
        artifact_id,
        &source.original_path,
    )?;
    let parent = stored_path
        .parent()
        .ok_or_else(|| "invalid export destination".to_string())?;
    fs::create_dir_all(parent)
        .map_err(|err| format!("failed to create artifact directory {:?}: {err}", parent))?;
    fs::copy(&source.original_path, &stored_path).map_err(|err| {
        format!(
            "failed to copy file {:?} to {:?}: {err}",
            source.original_path, stored_path
        )
    })?;

    Ok(ExportFileOutput {
        artifact_id: artifact_id.trim().to_string(),
        original_path: source.original_path,
        stored_path: stored_path.clone(),
        filename: source.filename.clone(),
        mime_type: guess_mime_type(&source.filename),
        bytes: source.bytes,
        sha256: hash_file(&stored_path)?,
    })
}

struct ExportSource {
    original_path: PathBuf,
    filename: String,
    bytes: usize,
}

fn resolve_export_source(
    config: &SandboxConfig,
    path: &str,
    max_bytes: Option<usize>,
) -> Result<ExportSource, String> {
    let original_path = resolve_read_path(path, config)?;
    let metadata = fs::metadata(&original_path)
        .map_err(|err| format!("failed to stat file {:?}: {err}", original_path))?;
    if !metadata.is_file() {
        return Err(format!("path is not a file: {}", original_path.display()));
    }

    let bytes = usize::try_from(metadata.len()).map_err(|_| {
        format!(
            "file is too large to export on this platform: {}",
            original_path.display()
        )
    })?;
    if let Some(limit) = max_bytes
        && bytes > limit
    {
        return Err(format!("file exceeds max_bytes limit ({limit} bytes)"));
    }

    let filename = original_path
        .file_name()
        .and_then(|value| value.to_str())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("failed to derive filename from {}", original_path.display()))?;

    Ok(ExportSource {
        original_path,
        filename,
        bytes,
    })
}

fn hash_file(path: &Path) -> Result<String, String> {
    let mut file = File::open(path)
        .map_err(|err| format!("failed to open exported file {:?}: {err}", path))?;
    let mut hasher = Sha256::new();
    let mut buffer = [0u8; 8192];
    loop {
        let read = file
            .read(&mut buffer)
            .map_err(|err| format!("failed to read exported file {:?}: {err}", path))?;
        if read == 0 {
            break;
        }
        hasher.update(&buffer[..read]);
    }
    Ok(format!("{:x}", hasher.finalize()))
}

fn build_export_destination(
    config: &SandboxConfig,
    artifact_root: &str,
    session_id: &str,
    artifact_id: &str,
    original_path: &Path,
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
