use crate::Response;
use crate::sandbox::SandboxConfig;
use crate::sandbox::file_tools::{
    apply_diff_impl, export_file_impl, list_files_impl, read_file_impl, search_files_impl,
};
use serde_json::{Value, json};

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "LIST_FILES" => Some(handle_list_files(params)),
        "READ_FILE" => Some(handle_read_file(params)),
        "SEARCH_FILES" => Some(handle_search_files(params)),
        "APPLY_DIFF" => Some(handle_apply_diff(params)),
        "EXPORT_FILE" => Some(handle_export_file(params)),
        _ => None,
    }
}

pub(crate) fn handle_list_files(params: &Value) -> Response {
    let path = params
        .get("path")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".");

    let result = match list_files_impl(&SandboxConfig::default(), path) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "path": path,
        "entries": result.entries,
    }))
}

pub(crate) fn handle_read_file(params: &Value) -> Response {
    let path = match parse_required_string(params, "path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let start_line = match parse_optional_usize(params, "start_line") {
        Ok(start_line) => start_line,
        Err(err) => return Response::error(err),
    };
    let end_line = match parse_optional_usize(params, "end_line") {
        Ok(end_line) => end_line,
        Err(err) => return Response::error(err),
    };

    let result = match read_file_impl(&SandboxConfig::default(), &path, start_line, end_line) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "path": result.path.display().to_string(),
        "requested_start_line": result.requested_start_line,
        "requested_end_line": result.requested_end_line,
        "returned_start_line": result.returned_start_line,
        "returned_end_line": result.returned_end_line,
        "total_lines": result.total_lines,
        "content": format_numbered_content(&result.content, result.returned_start_line),
    }))
}

pub(crate) fn handle_search_files(params: &Value) -> Response {
    let keyword = match parse_required_string(params, "keyword") {
        Ok(keyword) => keyword,
        Err(err) => return Response::error(err),
    };
    let dir_path = params
        .get("dir_path")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or(".")
        .to_string();
    let case_sensitive = match parse_optional_bool(params, "case_sensitive") {
        Ok(case_sensitive) => case_sensitive.unwrap_or(true),
        Err(err) => return Response::error(err),
    };

    let result = match search_files_impl(
        &SandboxConfig::default(),
        &keyword,
        &dir_path,
        case_sensitive,
    ) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "dir_path": result.dir_path.display().to_string(),
        "keyword": keyword,
        "case_sensitive": case_sensitive,
        "matches": result.matches,
    }))
}

pub(crate) fn handle_apply_diff(params: &Value) -> Response {
    let path = match parse_required_string(params, "path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let diff_text = match parse_required_string(params, "diff_text") {
        Ok(diff_text) => diff_text,
        Err(err) => return Response::error(err),
    };

    let result = match apply_diff_impl(&SandboxConfig::default(), &path, &diff_text) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "path": result.path.display().to_string(),
        "hunk_count": result.hunk_count,
        "added_lines": result.added_lines,
        "removed_lines": result.removed_lines,
        "hunk_ranges": result.hunk_ranges.iter().map(|hunk| {
            json!({
                "old_start": hunk.old_start,
                "old_count": hunk.old_count,
                "new_start": hunk.new_start,
                "new_count": hunk.new_count,
            })
        }).collect::<Vec<_>>(),
        "message": result.message,
    }))
}

pub(crate) fn handle_export_file(params: &Value) -> Response {
    let path = match parse_required_string(params, "path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let session_id = match parse_required_string(params, "session_id") {
        Ok(session_id) => session_id,
        Err(err) => return Response::error(err),
    };
    let artifact_root = match parse_required_string(params, "artifact_root") {
        Ok(artifact_root) => artifact_root,
        Err(err) => return Response::error(err),
    };
    let artifact_id = match parse_required_string(params, "artifact_id") {
        Ok(artifact_id) => artifact_id,
        Err(err) => return Response::error(err),
    };
    let max_bytes = match parse_optional_usize(params, "max_bytes") {
        Ok(max_bytes) => max_bytes,
        Err(err) => return Response::error(err),
    };

    let result = match export_file_impl(
        &SandboxConfig::default(),
        &path,
        &session_id,
        &artifact_root,
        &artifact_id,
        max_bytes,
    ) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "artifact_id": result.artifact_id,
        "filename": result.filename,
        "mime_type": result.mime_type,
        "bytes": result.bytes,
        "sha256": result.sha256,
        "stored_path": result.stored_path.display().to_string(),
        "original_path": result.original_path.display().to_string(),
    }))
}

fn parse_required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn parse_optional_usize(params: &Value, field: &str) -> Result<Option<usize>, String> {
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

fn parse_optional_bool(params: &Value, field: &str) -> Result<Option<bool>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    raw.as_bool()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a boolean"))
}

fn format_numbered_content(content: &str, start_line: usize) -> String {
    if content.is_empty() || start_line == 0 {
        return String::new();
    }

    let lines: Vec<&str> = content.lines().collect();
    let last_line = start_line + lines.len().saturating_sub(1);
    let width = last_line.to_string().len();

    lines
        .into_iter()
        .enumerate()
        .map(|(index, line)| format!("{:>width$} | {}", start_line + index, line, width = width))
        .collect::<Vec<_>>()
        .join("\n")
}

#[cfg(test)]
mod tests {
    use super::{
        dispatch_action, handle_apply_diff, handle_export_file, handle_list_files,
        handle_read_file, handle_search_files,
    };
    use serde_json::json;
    use std::{
        fs,
        path::PathBuf,
        time::{SystemTime, UNIX_EPOCH},
    };

    #[test]
    fn handle_read_file_returns_numbered_content() {
        let root = make_temp_dir();
        let file = root.join("sample.txt");
        fs::write(&file, "alpha\nbeta\ngamma\n").expect("write fixture");

        let response = handle_read_file(&json!({
            "path": file.to_string_lossy(),
            "start_line": 2,
            "end_line": 3
        }));

        assert_eq!(response.status, "success");
        assert_eq!(response.payload["requested_start_line"], 2);
        assert_eq!(response.payload["returned_start_line"], 2);
        let content = response.payload["content"]
            .as_str()
            .expect("content should be a string");
        assert!(content.contains("2 | beta"));
        assert!(content.contains("3 | gamma"));

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn handle_list_files_returns_sorted_entries_with_compatible_fields() {
        let root = make_temp_dir();
        fs::create_dir_all(root.join("docs")).expect("create docs");
        fs::write(root.join("z-last.txt"), "z").expect("write z-last.txt");
        fs::write(root.join("a-first.txt"), "a").expect("write a-first.txt");

        let response = handle_list_files(&json!({
            "path": root.to_string_lossy(),
        }));

        assert_eq!(response.status, "success");
        assert_eq!(response.payload["path"], root.to_string_lossy().to_string());
        assert_eq!(
            response.payload["entries"],
            json!(["a-first.txt", "docs/", "z-last.txt"])
        );

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn handle_list_files_blocks_sensitive_directory_names() {
        let root = make_temp_dir();
        let blocked = root.join("credentials-vault");
        fs::create_dir_all(&blocked).expect("create blocked dir");

        let response = handle_list_files(&json!({
            "path": blocked.to_string_lossy(),
        }));

        assert_eq!(response.status, "error");
        assert!(response.error.contains("sensitive file blocked"));

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn handle_search_files_returns_matches() {
        let root = make_temp_dir();
        let src = root.join("src");
        fs::create_dir_all(&src).expect("create src");
        fs::write(src.join("a.txt"), "TODO: first\nnoop\nTODO: second\n").expect("write a.txt");

        let response = handle_search_files(&json!({
            "keyword": "TODO",
            "dir_path": root.to_string_lossy(),
        }));

        assert_eq!(response.status, "success");
        let matches = response.payload["matches"]
            .as_array()
            .expect("matches should be an array");
        assert_eq!(matches.len(), 2);
        assert_eq!(matches[0], "src/a.txt:1:TODO: first");
        assert_eq!(matches[1], "src/a.txt:3:TODO: second");

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn handle_search_files_rejects_invalid_case_sensitive_type() {
        let response = handle_search_files(&json!({
            "keyword": "TODO",
            "case_sensitive": "yes"
        }));

        assert_eq!(response.status, "error");
        assert!(response.error.contains("case_sensitive must be a boolean"));
    }

    #[test]
    fn handle_apply_diff_updates_file() {
        let root = make_temp_dir();
        let file = root.join("patch.txt");
        fs::write(&file, "alpha\nbeta\ngamma\n").expect("write fixture");

        let response = handle_apply_diff(&json!({
            "path": file.to_string_lossy(),
            "diff_text": "@@ -1,3 +1,3 @@\n alpha\n-beta\n+beta2\n gamma\n"
        }));

        assert_eq!(response.status, "success");
        let updated = fs::read_to_string(&file).expect("read updated file");
        assert_eq!(updated, "alpha\nbeta2\ngamma\n");
        assert_eq!(response.payload["hunk_count"], 1);

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn handle_export_file_copies_file_into_session_artifact_directory() {
        let root = make_temp_dir();
        let source = root.join("notes.txt");
        let artifact_root = root.join("artifacts");
        fs::write(&source, "hello export\n").expect("write source file");

        let response = handle_export_file(&json!({
            "path": source.to_string_lossy(),
            "session_id": "session-1",
            "artifact_root": artifact_root.to_string_lossy(),
            "artifact_id": "artifact-1",
            "max_bytes": 1024
        }));

        assert_eq!(response.status, "success");
        assert_eq!(response.payload["artifact_id"], "artifact-1");
        assert_eq!(response.payload["filename"], "notes.txt");
        let stored_path = response.payload["stored_path"]
            .as_str()
            .expect("stored_path should be a string");
        assert!(stored_path.contains("artifacts/sessions/session-1/artifact-1.txt"));
        assert!(PathBuf::from(stored_path).exists());

        fs::remove_dir_all(root).ok();
    }

    #[test]
    fn dispatch_action_returns_none_for_unknown_file_action() {
        assert!(dispatch_action("PING", &json!({})).is_none());
    }

    fn make_temp_dir() -> PathBuf {
        let mut dir = std::env::current_dir().expect("resolve current dir");
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos();
        dir.push(format!(
            "ghost-os-native-file-actions-test-{}-{nanos}",
            std::process::id()
        ));
        fs::create_dir_all(&dir).expect("create temp dir");
        dir
    }
}
