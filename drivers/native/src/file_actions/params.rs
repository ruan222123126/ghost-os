use serde_json::Value;

pub(super) struct ListFilesRequest {
    pub(super) path: String,
}

pub(super) struct ReadFileRequest {
    pub(super) path: String,
    pub(super) start_line: Option<usize>,
    pub(super) end_line: Option<usize>,
}

pub(super) struct SearchFilesRequest {
    pub(super) keyword: String,
    pub(super) dir_path: String,
    pub(super) case_sensitive: bool,
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
    pub(super) fn parse(params: &Value) -> Self {
        let path = params
            .get("path")
            .and_then(Value::as_str)
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .unwrap_or(".");
        Self {
            path: path.to_string(),
        }
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

impl SearchFilesRequest {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        let dir_path = params
            .get("dir_path")
            .and_then(Value::as_str)
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .unwrap_or(".");

        Ok(Self {
            keyword: required_string(params, "keyword")?,
            dir_path: dir_path.to_string(),
            case_sensitive: optional_bool(params, "case_sensitive")?.unwrap_or(true),
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

fn required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn optional_usize(params: &Value, field: &str) -> Result<Option<usize>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_u64()
        .ok_or_else(|| format!("{field} must be a non-negative integer"))?;
    usize::try_from(value)
        .map(Some)
        .map_err(|_| format!("{field} is too large"))
}

fn optional_bool(params: &Value, field: &str) -> Result<Option<bool>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_bool()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a boolean"))
}
