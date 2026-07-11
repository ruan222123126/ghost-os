use super::{handle_mouse_move, resolve_target_point};
use crate::display_scale::record_display_scale;
use serde_json::json;

#[test]
fn handle_mouse_move_requires_xy() {
    let response = handle_mouse_move(&json!({"x": 10}));
    assert_eq!(response.status, "error");
    assert!(response.error.contains("y"));
}

#[test]
fn resolve_target_point_uses_display_scale() {
    record_display_scale(17, 2.0, 1.5);
    let (x, y, sx, sy) = resolve_target_point(200, 150, Some(17));
    assert_eq!((x, y), (100, 100));
    assert_eq!((sx, sy), (2.0, 1.5));
}
