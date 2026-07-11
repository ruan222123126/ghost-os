use glob::glob;
use std::env;
use std::ffi::OsString;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

use super::path_env::{dedupe_dirs, env_dir, home_dir, path_dirs};

const CODEX_NOT_FOUND_MESSAGE: &str = "codex executable not found; searched PATH, NODE_BIN_PATH sibling, and common Node/npm install directories; set codex_cli_path or GHOST_CODEX_CLI_PATH to a valid codex executable";
const NODE_NOT_FOUND_MESSAGE: &str = "node executable not found; searched GHOST_NODE_BIN_PATH override, NODE_BIN_PATH, PATH, and common Node install directories; set node_bin_path or GHOST_NODE_BIN_PATH to a valid node executable";

pub(crate) fn resolve_codex_executable(explicit_path: Option<&str>) -> Result<PathBuf, String> {
    if let Some(path) = normalize_optional_path(explicit_path) {
        return resolve_explicit_codex_path(&path);
    }
    if let Some(path) = find_codex_on_path() {
        return Ok(path);
    }
    if let Some(path) =
        find_named_executable_in_dirs(runtime_candidate_dirs(), codex_candidate_names())
    {
        return Ok(path);
    }
    Err(CODEX_NOT_FOUND_MESSAGE.to_string())
}

pub(crate) fn resolve_node_executable(explicit_path: Option<&str>) -> Result<PathBuf, String> {
    if let Some(path) = normalize_optional_path(explicit_path) {
        return resolve_explicit_node_path(&path);
    }
    if let Some(path) = node_bin_path_from_env() {
        return Ok(path);
    }
    if let Some(path) =
        find_named_executable_in_dirs(path_dirs(env::var_os("PATH")), node_candidate_names())
    {
        return Ok(path);
    }
    if let Some(path) = find_named_executable_in_dirs(common_node_dirs(), node_candidate_names()) {
        return Ok(path);
    }
    Err(NODE_NOT_FOUND_MESSAGE.to_string())
}

fn normalize_optional_path(input: Option<&str>) -> Option<PathBuf> {
    let trimmed = input?.trim();
    if trimmed.is_empty() {
        return None;
    }
    Some(PathBuf::from(trimmed))
}

fn resolve_explicit_codex_path(path: &Path) -> Result<PathBuf, String> {
    resolve_explicit_path(path, "codex", "codex_cli_path or GHOST_CODEX_CLI_PATH")
}

fn resolve_explicit_node_path(path: &Path) -> Result<PathBuf, String> {
    resolve_explicit_path(path, "node", "node_bin_path or GHOST_NODE_BIN_PATH")
}

fn resolve_explicit_path(path: &Path, kind: &str, hint: &str) -> Result<PathBuf, String> {
    if is_runnable_file(path) {
        return Ok(path.to_path_buf());
    }
    Err(format!(
        "configured {kind} executable path is not runnable: {}; set {hint} to a valid {kind} executable",
        path.display()
    ))
}

fn find_codex_on_path() -> Option<PathBuf> {
    find_named_executable_in_dirs(path_dirs(env::var_os("PATH")), codex_candidate_names())
}

fn runtime_candidate_dirs() -> Vec<PathBuf> {
    let mut dirs = Vec::new();
    if let Some(dir) = node_bin_dir_from_env() {
        dirs.push(dir);
    }
    dirs.extend(common_codex_dirs());
    dedupe_dirs(dirs)
}

fn node_bin_dir_from_env() -> Option<PathBuf> {
    let node_bin = env::var_os("NODE_BIN_PATH")?;
    codex_dir_from_node_bin_path(Path::new(&node_bin))
}

fn node_bin_path_from_env() -> Option<PathBuf> {
    node_bin_path_from_value(env::var_os("NODE_BIN_PATH"))
}

pub(crate) fn node_bin_path_from_value(value: Option<OsString>) -> Option<PathBuf> {
    let path = PathBuf::from(value?);
    if is_runnable_file(&path) {
        return Some(path);
    }
    None
}

fn common_codex_dirs() -> Vec<PathBuf> {
    let mut dirs = Vec::new();
    dirs.extend(unix_like_codex_dirs());
    dirs.extend(windows_codex_dirs());
    dedupe_dirs(dirs)
}

fn common_node_dirs() -> Vec<PathBuf> {
    let mut dirs = Vec::new();
    dirs.extend(unix_like_codex_dirs());
    dirs.extend(windows_node_dirs());
    dedupe_dirs(dirs)
}

#[cfg(not(target_os = "windows"))]
fn unix_like_codex_dirs() -> Vec<PathBuf> {
    let mut dirs = vec![PathBuf::from("/usr/local/bin"), PathBuf::from("/usr/bin")];
    #[cfg(target_os = "macos")]
    dirs.push(PathBuf::from("/opt/homebrew/bin"));
    if let Some(home) = home_dir() {
        dirs.extend(nvm_version_bin_dirs(&home));
        dirs.push(home.join(".volta/bin"));
        dirs.push(home.join(".asdf/shims"));
        dirs.push(home.join(".local/share/mise/shims"));
    }
    if let Some(nvm_bin) = env_dir("NVM_BIN") {
        dirs.push(nvm_bin);
    }
    if let Some(volta_home) = env_dir("VOLTA_HOME") {
        dirs.push(volta_home.join("bin"));
    }
    dedupe_dirs(dirs)
}

#[cfg(target_os = "windows")]
fn unix_like_codex_dirs() -> Vec<PathBuf> {
    Vec::new()
}

#[cfg(target_os = "windows")]
fn windows_codex_dirs() -> Vec<PathBuf> {
    let mut dirs = Vec::new();
    if let Some(appdata) = env_dir("APPDATA") {
        dirs.push(appdata.join("npm"));
    }
    if let Some(local) = env_dir("LOCALAPPDATA") {
        dirs.push(local.join("Volta").join("bin"));
    }
    if let Some(path) = env_dir("NVM_SYMLINK") {
        dirs.push(path);
    }
    if let Some(path) = env_dir("ProgramFiles") {
        dirs.push(path.join("nodejs"));
    }
    if let Some(path) = env_dir("ProgramFiles(x86)") {
        dirs.push(path.join("nodejs"));
    }
    dedupe_dirs(dirs)
}

#[cfg(not(target_os = "windows"))]
fn windows_codex_dirs() -> Vec<PathBuf> {
    Vec::new()
}

#[cfg(target_os = "windows")]
fn windows_node_dirs() -> Vec<PathBuf> {
    let mut dirs = Vec::new();
    if let Some(local) = env_dir("LOCALAPPDATA") {
        dirs.push(local.join("Volta").join("bin"));
    }
    if let Some(path) = env_dir("NVM_SYMLINK") {
        dirs.push(path);
    }
    if let Some(path) = env_dir("ProgramFiles") {
        dirs.push(path.join("nodejs"));
    }
    if let Some(path) = env_dir("ProgramFiles(x86)") {
        dirs.push(path.join("nodejs"));
    }
    dedupe_dirs(dirs)
}

#[cfg(not(target_os = "windows"))]
fn windows_node_dirs() -> Vec<PathBuf> {
    Vec::new()
}

pub(crate) fn codex_dir_from_node_bin_path(node_bin_path: &Path) -> Option<PathBuf> {
    node_bin_path.parent().map(Path::to_path_buf)
}

pub(crate) fn nvm_version_bin_dirs(home: &Path) -> Vec<PathBuf> {
    let pattern = home.join(".nvm/versions/node/*/bin");
    let Some(raw) = pattern.to_str() else {
        return Vec::new();
    };
    let Ok(paths) = glob(raw) else {
        return Vec::new();
    };
    paths.flatten().collect()
}

fn find_named_executable_in_dirs(dirs: Vec<PathBuf>, names: &[&str]) -> Option<PathBuf> {
    for dir in dirs {
        if let Some(path) = find_named_executable_in_dir(&dir, names) {
            return Some(path);
        }
    }
    None
}

fn find_named_executable_in_dir(dir: &Path, names: &[&str]) -> Option<PathBuf> {
    for name in names {
        let candidate = dir.join(name);
        if is_runnable_file(&candidate) {
            return Some(candidate);
        }
    }
    None
}

#[cfg(target_os = "windows")]
fn codex_candidate_names() -> &'static [&'static str] {
    &["codex.cmd", "codex.exe", "codex.bat", "codex"]
}

#[cfg(not(target_os = "windows"))]
fn codex_candidate_names() -> &'static [&'static str] {
    &["codex"]
}

#[cfg(target_os = "windows")]
fn node_candidate_names() -> &'static [&'static str] {
    &["node.exe", "node"]
}

#[cfg(not(target_os = "windows"))]
fn node_candidate_names() -> &'static [&'static str] {
    &["node"]
}

fn is_runnable_file(path: &Path) -> bool {
    let Ok(metadata) = path.metadata() else {
        return false;
    };
    if !metadata.is_file() {
        return false;
    }
    is_runnable_metadata(&metadata)
}

#[cfg(unix)]
fn is_runnable_metadata(metadata: &std::fs::Metadata) -> bool {
    metadata.permissions().mode() & 0o111 != 0
}

#[cfg(not(unix))]
fn is_runnable_metadata(_metadata: &std::fs::Metadata) -> bool {
    true
}

#[cfg(test)]
#[path = "executable_test.rs"]
mod tests;
