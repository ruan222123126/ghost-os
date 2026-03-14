use pyo3::prelude::*;
use serde_json::{json, to_string};

use crate::sandbox::tool_runtime::ToolRuntime;

use super::super::search::search_files_impl;

pub(crate) fn search_files_py(
    runtime: &ToolRuntime,
    keyword: String,
    dir_path: String,
    case_sensitive: bool,
) -> PyResult<Vec<String>> {
    let keyword = keyword.trim().to_string();
    let dir_path = normalize_path_arg(Some(&dir_path));
    let args = json!({
        "keyword": &keyword,
        "dir_path": &dir_path,
        "case_sensitive": case_sensitive,
    });

    match search_files_impl(runtime.config(), &keyword, &dir_path, case_sensitive) {
        Ok(result) => {
            runtime.log_success(
                "search_files",
                args,
                to_string(&result.matches).unwrap_or_default(),
            );
            Ok(result.matches)
        }
        Err(err) => Err(runtime.log_error("search_files", args, err)),
    }
}

fn normalize_path_arg(path: Option<&str>) -> String {
    path.map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".")
        .to_string()
}
