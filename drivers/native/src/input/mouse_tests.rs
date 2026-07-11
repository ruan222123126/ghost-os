#[cfg(target_os = "linux")]
use super::map_spawn_ydotool_error;
use super::{
    button_name, map_spawn_error, normalized_repeat, parse_click_button, resolve_target_point,
};
use crate::display_scale::record_display_scale;
use serde_json::json;

#[test]
fn parse_click_button_defaults_to_left() {
    let button = parse_click_button(&json!({})).expect("must parse");
    assert_eq!(button_name(button), "left");
}

#[test]
fn parse_click_button_supports_case_insensitive_middle() {
    let button =
        parse_click_button(&json!({"button":" MiDdLe "})).expect("must parse middle button");
    assert_eq!(button_name(button), "middle");
}

#[test]
fn parse_click_button_rejects_unsupported_values() {
    let err = parse_click_button(&json!({"button":"forward"})).expect_err("must fail");
    assert!(err.contains("button must be one of"));
}

#[test]
fn resolve_target_point_uses_cached_display_scale() {
    record_display_scale(7, 2.0, 1.5);
    let (x, y, sx, sy) = resolve_target_point(400, 300, Some(7));
    assert_eq!((x, y), (200, 200));
    assert_eq!((sx, sy), (2.0, 1.5));
}

#[test]
fn resolve_target_point_defaults_to_identity_when_display_unknown() {
    let (x, y, sx, sy) = resolve_target_point(123, 456, Some(777));
    assert_eq!((x, y), (123, 456));
    assert_eq!((sx, sy), (1.0, 1.0));
}

#[test]
fn normalized_repeat_never_returns_zero() {
    assert_eq!(normalized_repeat(0), 1);
    assert_eq!(normalized_repeat(3), 3);
}

#[test]
fn map_spawn_error_returns_not_found_hint() {
    let err = std::io::Error::from(std::io::ErrorKind::NotFound);
    let message = map_spawn_error("ydotool", err, Some("ydotool missing"));
    assert_eq!(message, "ydotool missing");
}

#[test]
fn map_spawn_error_returns_spawn_message_for_other_errors() {
    let err = std::io::Error::from(std::io::ErrorKind::PermissionDenied);
    let message = map_spawn_error("xdotool", err, None);
    assert!(message.contains("spawn xdotool failed"));
}

#[cfg(target_os = "linux")]
#[test]
fn map_spawn_ydotool_error_uses_wayland_specific_hint() {
    let err = std::io::Error::from(std::io::ErrorKind::NotFound);
    let message = map_spawn_ydotool_error(err);
    assert!(message.contains("required on Wayland"));
}
