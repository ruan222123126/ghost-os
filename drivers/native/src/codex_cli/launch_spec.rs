use std::env;
use std::ffi::OsString;
use std::fs;
use std::path::{Path, PathBuf};

use super::executable;
use super::path_env::path_dirs;

#[derive(Debug)]
pub(crate) struct CodexLaunchSpec {
    pub(crate) executable_path: PathBuf,
    pub(crate) path_override: Option<OsString>,
}

pub(crate) fn resolve_codex_launch_spec(
    codex_executable_path: Option<&str>,
    node_executable_path: Option<&str>,
) -> Result<CodexLaunchSpec, String> {
    let executable_path = executable::resolve_codex_executable(codex_executable_path)?;
    if !launcher_requires_node(&executable_path)? {
        return Ok(CodexLaunchSpec {
            executable_path,
            path_override: None,
        });
    }
    let node_path = executable::resolve_node_executable(node_executable_path)?;
    let node_dir = node_path.parent().ok_or_else(|| {
        format!(
            "node executable has no parent directory: {}",
            node_path.display()
        )
    })?;
    Ok(CodexLaunchSpec {
        executable_path,
        path_override: Some(prepend_path_dir(node_dir, env::var_os("PATH"))?),
    })
}

fn launcher_requires_node(path: &Path) -> Result<bool, String> {
    if is_cmd_launcher(path) {
        return Ok(true);
    }
    uses_env_node_shebang(path)
}

fn is_cmd_launcher(path: &Path) -> bool {
    let Some(extension) = path.extension().and_then(|value| value.to_str()) else {
        return false;
    };
    extension.eq_ignore_ascii_case("cmd") || extension.eq_ignore_ascii_case("bat")
}

fn uses_env_node_shebang(path: &Path) -> Result<bool, String> {
    let bytes = fs::read(path).map_err(|err| format!("read codex launcher failed: {err}"))?;
    let text = String::from_utf8_lossy(&bytes);
    let Some(line) = text.lines().next() else {
        return Ok(false);
    };
    let Some(shebang) = line.strip_prefix("#!") else {
        return Ok(false);
    };
    let normalized = shebang.trim();
    Ok(normalized.starts_with("/usr/bin/env node")
        || normalized.starts_with("env node")
        || normalized.starts_with("/usr/bin/env -S node")
        || normalized.starts_with("env -S node"))
}

pub(crate) fn prepend_path_dir(
    dir: &Path,
    current_path: Option<OsString>,
) -> Result<OsString, String> {
    let mut entries = vec![dir.to_path_buf()];
    for entry in path_dirs(current_path) {
        if entry != dir {
            entries.push(entry);
        }
    }
    env::join_paths(entries).map_err(|err| format!("join PATH failed: {err}"))
}

#[cfg(test)]
#[path = "launch_spec_test.rs"]
mod tests;
