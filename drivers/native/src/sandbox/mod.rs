// Sandbox module entrypoint; re-exports executor, restrictions, and tool adapters.

pub mod executor;
pub(crate) mod file_tools;
pub mod restrictions;
pub(crate) mod shell_tools;
pub(crate) mod tool_runtime;
pub mod tools;
pub(crate) mod web_tools;

mod diff_engine;
pub(crate) mod path_policy;
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
    pub max_search_matches: usize,
    pub max_search_files: usize,
    pub max_search_depth: usize,
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
            max_search_matches: 100,
            max_search_files: 2_000,
            max_search_depth: 25,
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
    let extra_reads =
        normalize_existing_dirs_optional(parse_env_path_list("GHOST_NATIVE_ALLOWED_READ_PATHS"));
    if !extra_reads.is_empty() {
        config.allowed_read_paths =
            merge_path_lists(config.allowed_read_paths.clone(), extra_reads);
    }

    let extra_writes =
        normalize_existing_dirs_optional(parse_env_path_list("GHOST_NATIVE_ALLOWED_WRITE_PATHS"));
    if !extra_writes.is_empty() {
        config.allowed_write_paths =
            merge_path_lists(config.allowed_write_paths.clone(), extra_writes);
    }
}

fn parse_env_path_list(var_name: &str) -> Vec<PathBuf> {
    let Ok(raw) = std::env::var(var_name) else {
        return Vec::new();
    };
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

pub use executor::PythonSandbox;

#[cfg(test)]
mod executor_test;
#[cfg(test)]
mod tools_test;

#[cfg(test)]
mod config_test {
    use super::{default_media_mounts, normalize_existing_dirs};
    use std::fs;
    use std::path::PathBuf;
    use std::time::{SystemTime, UNIX_EPOCH};

    fn make_temp_dir(prefix: &str) -> PathBuf {
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|duration| duration.as_nanos())
            .unwrap_or(0);
        let dir = std::env::temp_dir().join(format!(
            "ghost_os_sandbox_{prefix}_{}_{}",
            std::process::id(),
            nanos
        ));
        fs::create_dir_all(&dir).expect("create temp dir");
        dir
    }

    #[test]
    fn test_normalize_existing_dirs_canonicalizes_and_deduplicates() {
        let root = make_temp_dir("normalize");
        let nested = root.join("nested");
        fs::create_dir_all(&nested).expect("create nested dir");

        let paths = vec![
            root.clone(),
            nested.join(".."),
            PathBuf::from("/missing/ghost-os"),
        ];
        let normalized = normalize_existing_dirs(paths);

        assert_eq!(normalized.len(), 1);
        assert_eq!(normalized[0], root.to_string_lossy());

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn test_default_media_mounts_discovers_files_and_apps_dirs() {
        let media_root = make_temp_dir("media_root");
        let home_dir = media_root.join("home").join("demo");
        let fake_media = media_root.join("media").join("demo");
        fs::create_dir_all(&home_dir).expect("create home dir");
        fs::create_dir_all(fake_media.join("Files")).expect("create Files dir");
        fs::create_dir_all(fake_media.join("Apps")).expect("create Apps dir");

        let mounts = default_media_mounts_for_base(&home_dir, &media_root.join("media"));

        assert_eq!(mounts.len(), 2);
        assert!(
            mounts
                .iter()
                .any(|path| path.ends_with("/media/demo/Files"))
        );
        assert!(mounts.iter().any(|path| path.ends_with("/media/demo/Apps")));

        fs::remove_dir_all(media_root).ok();
    }

    fn default_media_mounts_for_base(
        home_dir: &std::path::Path,
        media_base: &std::path::Path,
    ) -> Vec<String> {
        let Some(user_name) = home_dir.file_name().and_then(|name| name.to_str()) else {
            return Vec::new();
        };

        ["Files", "Apps", "App"]
            .into_iter()
            .map(|name| media_base.join(user_name).join(name))
            .filter(|path| path.is_dir())
            .map(|path| path.to_string_lossy().to_string())
            .collect()
    }

    #[test]
    fn test_default_media_mounts_handles_unknown_user_dir() {
        let mounts = default_media_mounts(PathBuf::from("/").as_path());
        assert!(mounts.is_empty());
    }
}
