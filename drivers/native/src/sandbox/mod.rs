// Sandbox module entrypoint; re-exports executor, restrictions, and tool adapters.

#[cfg(feature = "python-sandbox")]
pub mod executor;
#[cfg(feature = "python-sandbox")]
pub(crate) mod executor_bootstrap;
pub(crate) mod file_tools;
#[cfg(feature = "python-sandbox")]
pub mod restrictions;
pub(crate) mod shell_tools;
#[cfg(feature = "python-sandbox")]
pub(crate) mod tool_runtime;
#[cfg(feature = "python-sandbox")]
pub mod tools;
#[cfg(feature = "python-sandbox")]
pub(crate) mod web_tools;

#[cfg(feature = "python-sandbox")]
pub(crate) const SCRIPT_EXEC_PRIVATE_TOOLS_NAME: &str = "__ghost_tools";

mod diff_engine;
pub(crate) mod path_policy;
#[cfg(feature = "python-sandbox")]
mod web_security;

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct SandboxConfig {
    pub allowed_modules: Vec<String>,
    pub default_script_timeout_ms: u64,
    pub max_script_timeout_ms: u64,
    pub default_script_memory_mb: u64,
    pub max_script_memory_mb: u64,
    pub default_shell_timeout_ms: u64,
    pub max_shell_timeout_ms: u64,
    pub max_shell_output_chars: usize,
    pub max_tool_log_chars: usize,
    pub max_file_read_lines: usize,
    pub max_file_write_bytes: usize,
    pub max_web_requests: usize,
    pub max_webpage_bytes: usize,
    pub allowed_read_paths: Vec<String>,
    pub allowed_write_paths: Vec<String>,
    pub blocked_patterns: Vec<String>,
}

impl Default for SandboxConfig {
    fn default() -> Self {
        let mut config = Self {
            allowed_modules: vec![
                "json".to_string(),
                "io".to_string(),
                "_io".to_string(),
                "re".to_string(),
                "math".to_string(),
                "base64".to_string(),
                "hashlib".to_string(),
                "urllib".to_string(),
                "csv".to_string(),
                "datetime".to_string(),
                "itertools".to_string(),
                "collections".to_string(),
                "functools".to_string(),
                "operator".to_string(),
                "string".to_string(),
                "time".to_string(),
                "os".to_string(),
                "sys".to_string(),
                "subprocess".to_string(),
            ],
            default_script_timeout_ms: 30_000,
            max_script_timeout_ms: 60_000,
            default_script_memory_mb: 256,
            max_script_memory_mb: 512,
            default_shell_timeout_ms: 15_000,
            max_shell_timeout_ms: 30_000,
            max_shell_output_chars: 2_000,
            max_tool_log_chars: 4_000,
            max_file_read_lines: 200,
            max_file_write_bytes: 1_048_576,
            max_web_requests: 10,
            max_webpage_bytes: 50 * 1024,
            allowed_read_paths: default_allowed_paths(),
            allowed_write_paths: default_allowed_paths(),
            blocked_patterns: vec![
                "*.env".to_string(),
                "*id_rsa*".to_string(),
                "*credentials*".to_string(),
            ],
        };
        merge_allowed_paths_from_env(&mut config);
        config
    }
}

fn default_allowed_paths() -> Vec<String> {
    let mut candidates = Vec::new();

    if let Ok(cwd) = std::env::current_dir() {
        candidates.push(cwd);
    }

    if let Some(home) = user_home_dir() {
        candidates.push(home.clone());
        candidates.extend(default_media_mounts(&home));
    }

    normalize_existing_dirs(candidates)
}

fn user_home_dir() -> Option<PathBuf> {
    std::env::var_os("HOME")
        .filter(|value| !value.is_empty())
        .map(PathBuf::from)
        .filter(|path| path.is_dir())
}

fn default_media_mounts(home_dir: &Path) -> Vec<PathBuf> {
    let Some(user_name) = home_dir.file_name().and_then(|name| name.to_str()) else {
        return Vec::new();
    };

    let media_root = PathBuf::from("/media").join(user_name);
    ["Files", "Apps", "App"]
        .into_iter()
        .map(|name| media_root.join(name))
        .filter(|path| path.is_dir())
        .collect()
}

fn normalize_existing_dirs(paths: Vec<PathBuf>) -> Vec<String> {
    let mut normalized = Vec::new();

    for path in paths {
        let Ok(canonical) = fs::canonicalize(&path) else {
            continue;
        };
        if !canonical.is_dir() {
            continue;
        }

        let as_string = canonical.to_string_lossy().to_string();
        if !normalized.iter().any(|existing| existing == &as_string) {
            normalized.push(as_string);
        }
    }

    if normalized.is_empty() {
        vec![".".to_string()]
    } else {
        normalized
    }
}

fn merge_allowed_paths_from_env(config: &mut SandboxConfig) {
    let extra_reads = normalize_existing_dirs_optional(parse_env_path_list_var(
        "GHOST_NATIVE_ALLOWED_READ_PATHS",
    ));
    if !extra_reads.is_empty() {
        config.allowed_read_paths =
            merge_path_lists(config.allowed_read_paths.clone(), extra_reads);
    }

    let extra_writes = normalize_existing_dirs_optional(parse_env_path_list_var(
        "GHOST_NATIVE_ALLOWED_WRITE_PATHS",
    ));
    if !extra_writes.is_empty() {
        config.allowed_write_paths =
            merge_path_lists(config.allowed_write_paths.clone(), extra_writes);
    }
}

fn parse_env_path_list_var(var_name: &str) -> Vec<PathBuf> {
    let Ok(raw) = std::env::var(var_name) else {
        return Vec::new();
    };
    parse_path_list_csv(&raw)
}

fn parse_path_list_csv(raw: &str) -> Vec<PathBuf> {
    raw.split(',')
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(PathBuf::from)
        .collect()
}

fn normalize_existing_dirs_optional(paths: Vec<PathBuf>) -> Vec<String> {
    let mut normalized = Vec::new();

    for path in paths {
        let Ok(canonical) = fs::canonicalize(&path) else {
            continue;
        };
        if !canonical.is_dir() {
            continue;
        }

        let as_string = canonical.to_string_lossy().to_string();
        if !normalized.iter().any(|existing| existing == &as_string) {
            normalized.push(as_string);
        }
    }
    normalized
}

fn merge_path_lists(mut base: Vec<String>, extra: Vec<String>) -> Vec<String> {
    for value in extra {
        if !base.iter().any(|existing| existing == &value) {
            base.push(value);
        }
    }
    if base.is_empty() {
        vec![".".to_string()]
    } else {
        base
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ExecutionResult {
    pub output: String,
    pub tool_calls_log: Vec<ToolCallLog>,
    pub error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct ToolCallLog {
    pub tool: String,
    pub args: serde_json::Value,
    pub result: String,
    pub error: Option<String>,
}

#[cfg(feature = "python-sandbox")]
pub use executor::PythonSandbox;

#[cfg(test)]
#[cfg(feature = "python-sandbox")]
mod executor_test;
#[cfg(test)]
#[cfg(feature = "python-sandbox")]
mod tools_test;

#[cfg(test)]
mod config_test;
#[cfg(test)]
mod diff_engine_test;
