use super::CapturedInput;

fn collect(bytes: &[u8]) -> Option<String> {
    let mut captured = CapturedInput::default();
    captured.push_bytes(bytes);
    captured.finish()
}

#[test]
fn collect_stops_on_newline_without_auto_submit() {
    assert_eq!(
        collect(b"hello world\nnext line"),
        Some("hello world".to_string())
    );
}

#[test]
fn collect_handles_backspace() {
    assert_eq!(collect(b"abcd\x7f\x7f12\r"), Some("ab12".to_string()));
}

#[test]
fn collect_ignores_ctrl_keys() {
    assert_eq!(collect(b"abc\x03\x04d\n"), Some("abcd".to_string()));
}

#[test]
fn collect_accepts_utf8_bytes() {
    assert_eq!(collect("你好\n".as_bytes()), Some("你好".to_string()));
}

#[test]
fn collect_filters_escape_sequences() {
    assert_eq!(collect(b"abc\x1b[A12\n"), Some("abc12".to_string()));
}
