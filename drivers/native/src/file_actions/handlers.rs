use serde_json::Value;
use serde_json::json;

use crate::Response;
use crate::sandbox::SandboxConfig;
use crate::sandbox::file_tools::{
    apply_diff_impl, list_files_impl, read_file_impl, search_files_impl, write_file_impl,
};

use super::format::format_numbered_content;
use super::params::{
    ApplyDiffRequest, ListFilesRequest, ReadFileRequest, SearchFilesRequest, WriteFileRequest,
};

pub(super) fn handle_list_files(params: &Value) -> Response {
    let request = match ListFilesRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match list_files_impl(&sandbox_config(), &request.path) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "path": request.path,
        "entries": result.entries,
    }))
}

pub(super) fn handle_search_files(params: &Value) -> Response {
    let request = match SearchFilesRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let matches = match search_files_impl(
        &sandbox_config(),
        &request.query,
        &request.path,
        request.max_results,
    ) {
        Ok(matches) => matches,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "matches": matches.iter().map(|item| {
            json!({
                "path": &item.path,
                "line": item.line,
                "text": &item.text,
            })
        }).collect::<Vec<_>>(),
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

pub(super) fn handle_write_file(params: &Value) -> Response {
    let request = match WriteFileRequest::parse(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let result = match write_file_impl(
        &sandbox_config(),
        &request.path,
        &request.content,
        &request.mode,
    ) {
        Ok(result) => result,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "message": result.message,
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

fn sandbox_config() -> SandboxConfig {
    SandboxConfig::default()
}
