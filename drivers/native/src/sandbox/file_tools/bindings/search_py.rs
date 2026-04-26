use pyo3::prelude::*;
use pyo3::types::PyDict;
use serde_json::{json, to_string};

use crate::sandbox::tool_runtime::ToolRuntime;

use super::super::search::{SearchMatchOutput, search_files_impl};

pub(crate) fn search_files_py(
    runtime: &ToolRuntime,
    py: Python<'_>,
    query: String,
    path: String,
    max_results: usize,
) -> PyResult<Vec<PyObject>> {
    let normalized_path = normalize_path_arg(&path);
    let args = json!({
        "query": query.trim(),
        "path": &normalized_path,
        "max_results": max_results,
    });

    match py.allow_threads(|| {
        search_files_impl(runtime.config(), &query, &normalized_path, max_results)
    }) {
        Ok(matches) => {
            let result_summary = to_string(&matches).unwrap_or_default();
            runtime.log_success("search_files", args, result_summary);
            matches_to_py_objects(py, &matches)
        }
        Err(err) => Err(runtime.log_error("search_files", args, err)),
    }
}

fn matches_to_py_objects(py: Python<'_>, matches: &[SearchMatchOutput]) -> PyResult<Vec<PyObject>> {
    let mut out = Vec::with_capacity(matches.len());
    for item in matches {
        let dict = PyDict::new_bound(py);
        dict.set_item("path", &item.path)?;
        dict.set_item("line", item.line)?;
        dict.set_item("text", &item.text)?;
        out.push(dict.into_py(py));
    }
    Ok(out)
}

fn normalize_path_arg(path: &str) -> String {
    let path = path.trim();
    if path.is_empty() {
        return ".".to_string();
    }
    path.to_string()
}
