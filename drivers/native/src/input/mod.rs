mod keyboard;
mod mouse;
mod mouse_drag;
mod mouse_scroll;
mod window_guard;

use crate::Response;
use serde_json::Value;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "TEXT_INPUT" => Some(keyboard::handle_text_input(params)),
        "KEY_HOTKEY" => Some(keyboard::handle_key_hotkey(params)),
        "MOUSE_CLICK" => Some(mouse::handle_mouse_click(params)),
        "MOUSE_DOUBLE_CLICK" => Some(mouse::handle_mouse_double_click(params)),
        "MOUSE_RIGHT_CLICK" => Some(mouse::handle_mouse_right_click(params)),
        "MOUSE_DRAG" => Some(mouse_drag::handle_mouse_drag(params)),
        "MOUSE_SCROLL" => Some(mouse_scroll::handle_mouse_scroll(params)),
        "ACTIVE_WINDOW_INFO" => Some(window_guard::handle_active_window_info(params)),
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
