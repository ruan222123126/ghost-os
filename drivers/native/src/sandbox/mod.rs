pub mod executor;
pub mod restrictions;
pub mod tools;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Deserialize)]
pub struct SandboxConfig {
    pub timeout_ms: u64,
    pub max_memory_mb: u64,
    pub allowed_modules: Vec<String>,
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
        Self {
            timeout_ms: 30_000,
            max_memory_mb: 256,
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
            max_file_read_lines: 200,
            max_file_write_bytes: 1_048_576,
            max_search_matches: 100,
            max_search_files: 2_000,
            max_search_depth: 25,
            max_web_requests: 10,
            max_webpage_bytes: 50 * 1024,
            allowed_read_paths: vec![".".to_string()],
            allowed_write_paths: vec![".".to_string()],
            blocked_patterns: vec![
                "*.env".to_string(),
                "*id_rsa*".to_string(),
                "*credentials*".to_string(),
            ],
        }
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
