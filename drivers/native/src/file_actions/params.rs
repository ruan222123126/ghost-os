use serde_json::Value;

use crate::json_params::{optional_string, optional_usize, required_string};
use crate::sandbox::file_tools::DEFAULT_SEARCH_MAX_RESULTS;

pub(super) struct ListFilesRequest {
    pub(super) path: String,
}

pub(super) struct SearchFilesRequest {
    pub(super) query: String,
    pub(super) path: String,
    pub(super) max_results: usize,
}

pub(super) struct ReadFileRequest {
    pub(super) path: String,
    pub(super) start_line: Option<usize>,
    pub(super) end_line: Option<usize>,
}

pub(super) struct WriteFileRequest {
    pub(super) path: String,
    pub(super) content: String,
    pub(super) mode: String,
}

pub(super) struct ApplyDiffRequest {
    pub(super) path: String,
    pub(super) diff_text: String,
}

pub(super) struct ExportFileRequest {
    pub(super) path: String,
    pub(super) session_id: String,
    pub(super) artifact_root: String,
    pub(super) artifact_id: String,
    pub(super) max_bytes: Option<usize>,
}

impl ListFilesRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        Ok(Self {
            path: optional_string(params, "path")?.unwrap_or_else(|| ".".to_string()),
        })
    }
}

impl SearchFilesRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        let path = optional_string(params, "path")?.unwrap_or_else(|| ".".to_string());
        let max_results =
            optional_usize(params, "max_results")?.unwrap_or(DEFAULT_SEARCH_MAX_RESULTS);
        Ok(Self {
            query: required_string(params, "query")?,
            path,
            max_results,
        })
    }
}

impl ReadFileRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        Ok(Self {
            path: required_string(params, "path")?,
            start_line: optional_usize(params, "start_line")?,
            end_line: optional_usize(params, "end_line")?,
        })
    }
}

impl WriteFileRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        let mode = optional_text(params, "mode")?.unwrap_or_else(|| "write".to_string());
        Ok(Self {
            path: required_string(params, "path")?,
            content: required_present_string(params, "content")?,
            mode,
        })
    }
}

impl ApplyDiffRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        Ok(Self {
            path: required_string(params, "path")?,
            diff_text: required_string(params, "diff_text")?,
        })
    }
}

impl ExportFileRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        Ok(Self {
            path: required_string(params, "path")?,
            session_id: required_string(params, "session_id")?,
            artifact_root: required_string(params, "artifact_root")?,
            artifact_id: required_string(params, "artifact_id")?,
            max_bytes: optional_usize(params, "max_bytes")?,
        })
    }
}

fn required_present_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn optional_text(params: &Value, field: &str) -> Result<Option<String>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_str()
        .map(|value| Some(value.to_string()))
        .ok_or_else(|| format!("{field} must be a string"))
}
