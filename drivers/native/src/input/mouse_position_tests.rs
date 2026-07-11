use super::{RawMousePosition, parse_xdotool_mouse_position, resolve_mouse_position_payload};

#[cfg(target_os = "linux")]
#[test]
fn parse_xdotool_mouse_position_reads_shell_output() {
    let position =
        parse_xdotool_mouse_position("X=120\nY=340\nSCREEN=0\nWINDOW=0\n").expect("must parse");
    assert_eq!(position, RawMousePosition { x: 120, y: 340 });
}

#[cfg(target_os = "linux")]
#[test]
fn parse_xdotool_mouse_position_rejects_missing_y() {
    let err = parse_xdotool_mouse_position("X=120\n").expect_err("must fail");
    assert!(err.contains("missing X or Y"));
}

#[test]
fn resolve_mouse_position_payload_keeps_raw_point_when_monitor_is_missing() {
    let payload = resolve_mouse_position_payload(RawMousePosition {
        x: i32::MIN,
        y: i32::MIN,
    });
    assert_eq!(payload.x, i32::MIN);
    assert_eq!(payload.y, i32::MIN);
    assert_eq!(payload.display_id, None);
}
