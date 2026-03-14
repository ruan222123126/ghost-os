use pyo3::PyErr;
use pyo3::exceptions::PyRuntimeError;
use serde_json::Value;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};

use super::{SandboxConfig, ToolCallLog};

#[cfg(test)]
pub(crate) const SCRIPT_SANDBOX_ALLOWED_TOOLS: &[&str] = &[
    "bash_exec",
    "list_files",
    "read_file",
    "write_file",
    "apply_diff",
    "search_files",
    "fetch_webpage",
];

#[derive(Clone)]
pub(crate) struct ToolRuntime {
    tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>,
    config: SandboxConfig,
    web_request_count: Arc<AtomicUsize>,
}

impl ToolRuntime {
    pub(crate) fn new(tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>, config: SandboxConfig) -> Self {
        Self {
            tool_calls_log,
            config,
            web_request_count: Arc::new(AtomicUsize::new(0)),
        }
    }

    pub(crate) fn config(&self) -> &SandboxConfig {
        &self.config
    }

    pub(crate) fn log_success(&self, tool: &str, args: Value, result: impl Into<String>) {
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: tool.to_string(),
                args,
                result: truncate_log_result(&self.config, &result.into()),
                error: None,
            },
        );
    }

    pub(crate) fn log_failure(
        &self,
        tool: &str,
        args: Value,
        result: impl Into<String>,
        err: impl Into<String>,
    ) -> PyErr {
        let err = err.into();
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: tool.to_string(),
                args,
                result: truncate_log_result(&self.config, &result.into()),
                error: Some(err.clone()),
            },
        );
        PyRuntimeError::new_err(err)
    }

    pub(crate) fn log_error(&self, tool: &str, args: Value, err: impl Into<String>) -> PyErr {
        let err = err.into();
        push_tool_log(
            &self.tool_calls_log,
            ToolCallLog {
                tool: tool.to_string(),
                args,
                result: String::new(),
                error: Some(err.clone()),
            },
        );
        PyRuntimeError::new_err(err)
    }

    pub(crate) fn consume_web_request_budget(&self) -> Result<(), String> {
        let request_number = self.web_request_count.fetch_add(1, Ordering::SeqCst) + 1;
        if request_number > self.config.max_web_requests {
            return Err(format!(
                "rate limit: max {} web requests per script",
                self.config.max_web_requests
            ));
        }
        Ok(())
    }
}

pub(crate) fn truncate_chars(text: &str, max_chars: usize, label: &str) -> String {
    if max_chars == 0 {
        return String::new();
    }

    let char_count = text.chars().count();
    if char_count <= max_chars {
        return text.to_string();
    }

    let truncated: String = text.chars().take(max_chars).collect();
    format!(
        "{}\n\n[... {}, {} more chars]",
        truncated,
        label,
        char_count - max_chars
    )
}

fn truncate_log_result(config: &SandboxConfig, text: &str) -> String {
    truncate_chars(text, config.max_tool_log_chars, "log truncated")
}

fn push_tool_log(tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>, log: ToolCallLog) {
    if let Ok(mut logs) = tool_calls_log.lock() {
        logs.push(log);
    }
}
