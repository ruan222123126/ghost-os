mod keyboard;
mod mouse;
mod window_guard;

use crate::Response;
use serde_json::Value;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "TEXT_INPUT" => Some(keyboard::handle_text_input(params)),
        "MOUSE_CLICK" => Some(mouse::handle_mouse_click(params)),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::dispatch_action;
    use serde_json::json;

    #[test]
    fn dispatch_action_returns_none_for_unknown_input_action() {
        assert!(dispatch_action("PING", &json!({})).is_none());
    }
}
