mod capture;
mod image_ops;
mod match_engine;
mod ocr;
mod params;
mod template_match;
pub(crate) mod types;

use crate::Response;
use serde_json::Value;

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "SCREEN_CAPTURE" => Some(capture::handle_screen_capture(params)),
        "IMAGE_CROP" => Some(image_ops::handle_image_crop(params)),
        "OCR_IMAGE" => Some(ocr::handle_ocr_image(params)),
        "TEMPLATE_MATCH_IMAGE" => Some(template_match::handle_template_match_image(params)),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::dispatch_action;
    use serde_json::json;

    #[test]
    fn dispatch_action_returns_none_for_unknown_screen_action() {
        assert!(dispatch_action("PING", &json!({})).is_none());
    }
}
