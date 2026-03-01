// Path policy helpers that enforce allowlist/blocklist checks for sandbox tool file access.

use glob::Pattern;
use std::fs;
use std::path::{Path, PathBuf};

use super::SandboxConfig;

pub(crate) fn resolve_read_path(path: &str, config: &SandboxConfig) -> Result<PathBuf, String> {
    let canonical = canonicalize_for_read(path)?;
    ensure_path_allowed(&canonical, &config.allowed_read_paths)?;
    ensure_not_blocked(&canonical, &config.blocked_patterns)?;
    Ok(canonical)
}

pub(crate) fn resolve_write_path(path: &str, config: &SandboxConfig) -> Result<PathBuf, String> {
    let canonical = canonicalize_for_write(path)?;
    ensure_path_allowed(&canonical, &config.allowed_write_paths)?;
    ensure_not_blocked(&canonical, &config.blocked_patterns)?;
    Ok(canonical)
}

pub(crate) fn is_blocked_path(path: &Path, blocked_patterns: &[String]) -> bool {
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
    let canonical_allowed = canonicalize_allowed_dirs(allowed_dirs)?;
    if canonical_allowed
        .iter()
        .any(|allowed| path.starts_with(allowed))
    {
        return Ok(());
    }

    Err("path not in allowed directories".to_string())
}

fn canonicalize_allowed_dirs(allowed_dirs: &[String]) -> Result<Vec<PathBuf>, String> {
    let mut canonical_allowed = Vec::with_capacity(allowed_dirs.len());
    for dir in allowed_dirs {
        let absolute = to_absolute_path(dir)?;
        let canonical = fs::canonicalize(&absolute)
            .map_err(|err| format!("invalid allowed directory {}: {err}", absolute.display()))?;
        canonical_allowed.push(canonical);
    }

    if canonical_allowed.is_empty() {
        return Err("no allowed directories configured".to_string());
    }

    Ok(canonical_allowed)
}

fn ensure_not_blocked(path: &Path, blocked_patterns: &[String]) -> Result<(), String> {
    if is_blocked_path(path, blocked_patterns) {
        return Err("access to sensitive file blocked".to_string());
    }
    Ok(())
}
