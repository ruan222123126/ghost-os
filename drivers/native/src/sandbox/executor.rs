use pyo3::prelude::*;
use pyo3::types::PyDict;
use std::sync::{Arc, Mutex, OnceLock};

use super::executor_bootstrap::{
    STDOUT_SETUP_CODE, build_helper_bootstrap as render_helper_bootstrap,
};
use super::restrictions::{setup_restricted_imports, validate_script_safety};
use super::tools::ToolsProxy;
use super::{ExecutionResult, SCRIPT_EXEC_PRIVATE_TOOLS_NAME, SandboxConfig, ToolCallLog};

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
        if let Err(err) = bind_private_tools(py, &locals, &tool_calls_log, config) {
            return execution_error(String::new(), &tool_calls_log, err);
        }
        if let Err(err) = setup_stdout_capture(py, &locals) {
            return execution_error(String::new(), &tool_calls_log, err);
        }

        let restrictions = match setup_restricted_imports(py, &config.allowed_modules, &locals) {
            Ok(guard) => guard,
            Err(err) => {
                restore_stdout_capture(py, &locals);
                return execution_error(
                    String::new(),
                    &tool_calls_log,
                    format!("failed to setup sandbox restrictions: {err}"),
                );
            }
        };
        if let Err(err) = install_helper_bootstrap(py, &locals) {
            let _ = restrictions.restore(py);
            restore_stdout_capture(py, &locals);
            return execution_error(String::new(), &tool_calls_log, err);
        }

        let run_result = py.run_bound(script, Some(&locals), Some(&locals));
        let output = captured_stdout(py, &locals);

        let _ = restrictions.restore(py);
        restore_stdout_capture(py, &locals);

        if let Err(err) = run_result {
            return execution_error(
                output,
                &tool_calls_log,
                format!("script execution error: {err}"),
            );
        }

        ExecutionResult {
            output,
            tool_calls_log: snapshot_tool_calls_log(&tool_calls_log),
            error: None,
        }
    })
}

fn bind_private_tools(
    py: Python<'_>,
    locals: &Bound<'_, PyDict>,
    tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>,
    config: &SandboxConfig,
) -> Result<(), String> {
    let tools_proxy = Py::new(py, ToolsProxy::new(tool_calls_log.clone(), config.clone()))
        .map_err(|err| format!("failed to create internal tools proxy: {err}"))?;
    locals
        .set_item(SCRIPT_EXEC_PRIVATE_TOOLS_NAME, tools_proxy)
        .map_err(|err| format!("failed to bind internal tools proxy: {err}"))
}

fn setup_stdout_capture(py: Python<'_>, locals: &Bound<'_, PyDict>) -> Result<(), String> {
    py.run_bound(STDOUT_SETUP_CODE, Some(locals), Some(locals))
        .map_err(|err| format!("failed to setup stdout capture: {err}"))
}

fn install_helper_bootstrap(py: Python<'_>, locals: &Bound<'_, PyDict>) -> Result<(), String> {
    py.run_bound(&build_helper_bootstrap(), Some(locals), Some(locals))
        .map_err(|err| format!("failed to install sandbox helpers: {err}"))
}

fn build_helper_bootstrap() -> String {
    render_helper_bootstrap(SCRIPT_EXEC_PRIVATE_TOOLS_NAME)
}

fn captured_stdout(py: Python<'_>, locals: &Bound<'_, PyDict>) -> String {
    py.eval_bound(
        "__ghost_stdout_capture.getvalue()",
        Some(locals),
        Some(locals),
    )
    .and_then(|value| value.extract::<String>())
    .unwrap_or_default()
}

fn restore_stdout_capture(py: Python<'_>, locals: &Bound<'_, PyDict>) {
    let _ = py.run_bound(
        "import sys; sys.stdout = __ghost_old_stdout",
        Some(locals),
        Some(locals),
    );
}

fn execution_error(
    output: String,
    tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>,
    error: String,
) -> ExecutionResult {
    ExecutionResult {
        output,
        tool_calls_log: snapshot_tool_calls_log(tool_calls_log),
        error: Some(error),
    }
}

fn snapshot_tool_calls_log(tool_calls_log: &Arc<Mutex<Vec<ToolCallLog>>>) -> Vec<ToolCallLog> {
    tool_calls_log
        .lock()
        .map(|logs| logs.clone())
        .unwrap_or_default()
}
