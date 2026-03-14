use serde_json::Value;
use serde_json::json;

use crate::Response;
use crate::sandbox::SandboxConfig;
use crate::sandbox::file_tools::{
    apply_diff_impl, export_file_impl, list_files_impl, read_file_impl, search_files_impl,
};

use super::format::format_numbered_content;
use super::params::{
    ApplyDiffRequest, ExportFileRequest, ListFilesRequest, ReadFileRequest, SearchFilesRequest,
};

pub(super) fn handle_list_files(params: &Value) -> Response {
    let request = ListFilesRequest::parse(params);
    let result = match list_files_impl(&sandbox_config(), &request.path) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "path": request.path,
        "entries": result.entries,
    }))
}

pub(super) fn handle_read_file(params: &Value) -> Response {
    let request = match ReadFileRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match read_file_impl(
        &sandbox_config(),
        &request.path,
        request.start_line,
        request.end_line,
    ) {
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

pub(super) fn handle_search_files(params: &Value) -> Response {
    let request = match SearchFilesRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match search_files_impl(
        &sandbox_config(),
        &request.keyword,
        &request.dir_path,
        request.case_sensitive,
    ) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "dir_path": result.dir_path.display().to_string(),
        "keyword": request.keyword,
        "case_sensitive": request.case_sensitive,
        "matches": result.matches,
    }))
}

pub(super) fn handle_apply_diff(params: &Value) -> Response {
    let request = match ApplyDiffRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match apply_diff_impl(&sandbox_config(), &request.path, &request.diff_text) {
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

pub(super) fn handle_export_file(params: &Value) -> Response {
    let request = match ExportFileRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match export_file_impl(
        &sandbox_config(),
        &request.path,
        &request.session_id,
        &request.artifact_root,
        &request.artifact_id,
        request.max_bytes,
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

fn sandbox_config() -> SandboxConfig {
    SandboxConfig::default()
}
