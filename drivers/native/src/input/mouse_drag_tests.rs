use super::{drag_duration, map_xdotool_spawn_error, parse_drag_request, resolve_drag_point};
use crate::display_scale::record_display_scale;
use serde_json::json;
use std::time::Duration;

#[test]
fn parse_drag_request_defaults_duration() {
    let request = parse_drag_request(&json!({
        "start_x": 1,
        "start_y": 2,
        "end_x": 3,
        "end_y": 4
    }))
    .expect("must parse");
    assert_eq!(request.duration_ms, 150);
}

#[test]
fn parse_drag_request_rejects_negative_duration() {
    let result = parse_drag_request(&json!({
        "start_x": 1,
        "start_y": 2,
        "end_x": 3,
        "end_y": 4,
        "duration_ms": -1
    }));
    assert!(result.is_err(), "must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("non-negative"));
}

#[test]
fn parse_drag_request_requires_start_coordinates() {
    let result = parse_drag_request(&json!({
        "start_y": 2,
        "end_x": 3,
        "end_y": 4
    }));
    assert!(result.is_err(), "must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("start_x"));
}

#[test]
fn resolve_drag_point_uses_display_scale() {
    record_display_scale(9, 2.0, 2.0);
    let (x, y, sx, sy) = resolve_drag_point(200, 100, Some(9));
    assert_eq!((x, y), (100, 50));
    assert_eq!((sx, sy), (2.0, 2.0));
}

#[test]
fn resolve_drag_point_defaults_to_identity_when_display_unknown() {
    let (x, y, sx, sy) = resolve_drag_point(11, 22, Some(777));
    assert_eq!((x, y), (11, 22));
    assert_eq!((sx, sy), (1.0, 1.0));
}

#[test]
fn drag_duration_clamps_negative_to_zero() {
    assert_eq!(drag_duration(-50), Duration::from_millis(0));
    assert_eq!(drag_duration(25), Duration::from_millis(25));
}

#[test]
fn map_xdotool_spawn_error_returns_not_found_message() {
    let err = std::io::Error::from(std::io::ErrorKind::NotFound);
    assert_eq!(
        map_xdotool_spawn_error(err),
        "xdotool is not installed".to_string()
    );
}

#[test]
fn map_xdotool_spawn_error_keeps_other_spawn_failures() {
    let err = std::io::Error::from(std::io::ErrorKind::PermissionDenied);
    let msg = map_xdotool_spawn_error(err);
    assert!(msg.contains("spawn xdotool failed"));
}
