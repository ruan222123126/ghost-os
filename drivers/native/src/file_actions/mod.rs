mod format;
mod handlers;
mod params;

#[cfg(test)]
mod tests;

use serde_json::Value;

use crate::Response;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "LIST_FILES" => Some(handlers::handle_list_files(params)),
        "READ_FILE" => Some(handlers::handle_read_file(params)),
        "SEARCH_FILES" => Some(handlers::handle_search_files(params)),
        "APPLY_DIFF" => Some(handlers::handle_apply_diff(params)),
        "EXPORT_FILE" => Some(handlers::handle_export_file(params)),
        _ => None,
    }
}
