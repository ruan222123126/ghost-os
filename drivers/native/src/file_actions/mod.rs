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
        "SEARCH_FILES" => Some(handlers::handle_search_files(params)),
        "READ_FILE" => Some(handlers::handle_read_file(params)),
        "WRITE_FILE" => Some(handlers::handle_write_file(params)),
        "APPLY_DIFF" => Some(handlers::handle_apply_diff(params)),
        _ => None,
    }
}
