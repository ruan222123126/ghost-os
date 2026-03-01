use serde_json::{Value, json};

use crate::{
    Response, browser_query, handle_bash_exec, handle_list_files, handle_mouse_click,
    handle_screen_shot, script_exec,
};

// dispatch_action 统一处理协议 action 到具体原子能力的映射。
pub(crate) fn dispatch_action(action: &str, params: &Value, trace_id: &str) -> Response {
    match action {
        "PING" => Response::success(json!({
            "message": "PONG",
            "trace_id": trace_id
        })),
        "LIST_FILES" => handle_list_files(params),
        "BASH_EXEC" => handle_bash_exec(params),
        "SCRIPT_EXEC" => script_exec::handle_script_exec(params),
        "SCREEN_SHOT" => handle_screen_shot(params),
        "MOUSE_CLICK" => handle_mouse_click(params),
        "BROWSER_QUERY" => browser_query::handle_browser_query(params),
        _ => Response::error(format!("unsupported action: {action}")),
    }
}
