use pyo3::prelude::*;
use pyo3::types::PyDict;
use std::sync::{Arc, Mutex, OnceLock};

use super::restrictions::{setup_restricted_imports, validate_script_safety};
use super::tools::ToolsProxy;
use super::{ExecutionResult, SandboxConfig, ToolCallLog};

pub struct PythonSandbox {
    config: SandboxConfig,
    tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>,
}

impl PythonSandbox {
    pub fn new(config: SandboxConfig) -> Self {
        Self {
            config,
            tool_calls_log: Arc::new(Mutex::new(Vec::new())),
        }
    }

    pub fn execute_blocking(&self, script: &str) -> ExecutionResult {
        if let Err(err) = validate_script_safety(script) {
            return ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(err),
            };
        }

        if let Ok(mut logs) = self.tool_calls_log.lock() {
            logs.clear();
        }

        execute_script_blocking(script, &self.config, self.tool_calls_log.clone())
    }
}

fn execute_script_blocking(
    script: &str,
    config: &SandboxConfig,
    tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>,
) -> ExecutionResult {
    // CPython builtins/import hooks are process-global; serialize script execution.
    static EXECUTION_LOCK: OnceLock<Mutex<()>> = OnceLock::new();
    let execution_lock = EXECUTION_LOCK.get_or_init(|| Mutex::new(()));
    let _serial_guard = execution_lock
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner());

    Python::with_gil(|py| {
        let locals = PyDict::new_bound(py);
        let tools_proxy = match Py::new(py, ToolsProxy::new(tool_calls_log.clone(), config.clone()))
        {
            Ok(proxy) => proxy,
            Err(err) => {
                return ExecutionResult {
                    output: String::new(),
                    tool_calls_log: Vec::new(),
                    error: Some(format!("failed to create tools proxy: {err}")),
                };
            }
        };
        if let Err(err) = locals.set_item("tools", tools_proxy) {
            return ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(format!("failed to bind tools proxy: {err}")),
            };
        }

        let stdout_setup = r#"
import io
import sys
__ghost_old_stdout = sys.stdout
__ghost_stdout_capture = io.StringIO()
sys.stdout = __ghost_stdout_capture
"#;
        if let Err(err) = py.run_bound(stdout_setup, None, Some(&locals)) {
            return ExecutionResult {
                output: String::new(),
                tool_calls_log: Vec::new(),
                error: Some(format!("failed to setup stdout capture: {err}")),
            };
        }

        let restrictions = match setup_restricted_imports(py, &config.allowed_modules) {
            Ok(guard) => guard,
            Err(err) => {
                let _ = py.run_bound(
                    "import sys; sys.stdout = __ghost_old_stdout",
                    None,
                    Some(&locals),
                );
                return ExecutionResult {
                    output: String::new(),
                    tool_calls_log: Vec::new(),
                    error: Some(format!("failed to setup sandbox restrictions: {err}")),
                };
            }
        };

        let run_result = py.run_bound(script, None, Some(&locals));

        let output = py
            .eval_bound("__ghost_stdout_capture.getvalue()", None, Some(&locals))
            .and_then(|value| value.extract::<String>())
            .unwrap_or_default();

        let _ = restrictions.restore(py);
        let _ = py.run_bound(
            "import sys; sys.stdout = __ghost_old_stdout",
            None,
            Some(&locals),
        );

        if let Err(err) = run_result {
            return ExecutionResult {
                output,
                tool_calls_log: snapshot_tool_calls_log(&tool_calls_log),
                error: Some(format!("script execution error: {err}")),
            };
        }

        ExecutionResult {
            output,
            tool_calls_log: snapshot_tool_calls_log(&tool_calls_log),
            error: None,
        }
    })
}

fn snapshot_tool_calls_log(tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>) -> Vec<ToolCallLog> {
    tool_calls_log
        .lock()
        .map(|logs| logs.clone())
        .unwrap_or_default()
}
