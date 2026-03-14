#![allow(clippy::useless_conversion)]

use pyo3::prelude::*;
use std::sync::{Arc, Mutex};

use super::file_tools::{
    apply_diff_py, list_files_py, read_file_py, search_files_py, write_file_py,
};
use super::shell_tools::bash_exec_py;
use super::tool_runtime::ToolRuntime;
use super::web_tools::fetch_webpage_py;
use super::{SandboxConfig, ToolCallLog};

#[pyclass]
pub struct ToolsProxy {
    runtime: ToolRuntime,
}

impl ToolsProxy {
    pub fn new(tool_calls_log: Arc<Mutex<Vec<ToolCallLog>>>, config: SandboxConfig) -> Self {
        Self {
            runtime: ToolRuntime::new(tool_calls_log, config),
        }
    }
}

#[allow(unsafe_op_in_unsafe_fn)]
// `#[pymethods]` expands these wrappers with redundant `PyErr` conversions that clippy flags.
#[pymethods]
impl ToolsProxy {
    #[pyo3(signature = (*, command))]
    fn bash_exec(&self, py: Python<'_>, command: String) -> PyResult<String> {
        bash_exec_py(&self.runtime, py, command)
    }

    #[pyo3(signature = (*, path=None))]
    fn list_files(&self, path: Option<String>) -> PyResult<Vec<String>> {
        list_files_py(&self.runtime, path)
    }

    #[pyo3(signature = (*, path, start_line=None, end_line=None))]
    fn read_file(
        &self,
        path: String,
        start_line: Option<usize>,
        end_line: Option<usize>,
    ) -> PyResult<String> {
        read_file_py(&self.runtime, path, start_line, end_line)
    }

    #[pyo3(signature = (*, path, content, mode="write"))]
    fn write_file(&self, path: String, content: String, mode: &str) -> PyResult<String> {
        write_file_py(&self.runtime, path, content, mode)
    }

    #[pyo3(signature = (*, path, diff_text))]
    fn apply_diff(&self, path: String, diff_text: String) -> PyResult<String> {
        apply_diff_py(&self.runtime, path, diff_text)
    }

    #[pyo3(signature = (*, keyword, dir_path=".".to_string(), case_sensitive=true))]
    fn search_files(
        &self,
        keyword: String,
        dir_path: String,
        case_sensitive: bool,
    ) -> PyResult<Vec<String>> {
        search_files_py(&self.runtime, keyword, dir_path, case_sensitive)
    }

    #[pyo3(signature = (*, url))]
    fn fetch_webpage(&self, py: Python<'_>, url: String) -> PyResult<String> {
        fetch_webpage_py(&self.runtime, py, url)
    }
}
