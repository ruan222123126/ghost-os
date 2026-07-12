mod format;
mod handlers;
mod params;

#[cfg(test)]
mod tests;

use serde_json::Value;

use crate::Response;
use crate::sandbox::SandboxConfig;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    let config = SandboxConfig::default();
    dispatch_action_with_config(action, params, &config)
}

fn dispatch_action_with_config(
    action: &str,
    params: &Value,
    config: &SandboxConfig,
) -> Option<Response> {
    match action {
        "LIST_FILES" => Some(handlers::handle_list_files(params, config)),
        "SEARCH_FILES" => Some(handlers::handle_search_files(params, config)),
        "READ_FILE" => Some(handlers::handle_read_file(params, config)),
        "WRITE_FILE" => Some(handlers::handle_write_file(params, config)),
        "APPLY_DIFF" => Some(handlers::handle_apply_diff(params, config)),
        _ => None,
    }
}
