#[cfg(target_os = "linux")]
use super::finalize_restore_result;
#[cfg(target_os = "linux")]
use super::keyboard_x11::{
    XdotoolTextToken, parse_active_window_id_output, tokenize_text_for_xdotool,
};
use super::{handle_text_input, normalize_hotkey_key, normalize_input_text, parse_hotkey_keys};
use serde_json::json;

#[test]
fn normalize_input_text_flattens_crlf() {
    assert_eq!(normalize_input_text("a\r\nb\rc"), "a\nb\nc");
}

#[test]
fn handle_text_input_rejects_empty_string() {
    let response = handle_text_input(&json!({"text":""}));
    assert_eq!(response.status, "error");
    assert_eq!(response.error, "text is required");
}

#[test]
fn normalize_hotkey_key_maps_common_modifiers() {
    assert_eq!(normalize_hotkey_key("CTRL").expect("must parse"), "ctrl");
    assert_eq!(normalize_hotkey_key("enter").expect("must parse"), "Return");
}

#[test]
fn parse_hotkey_keys_requires_array() {
    let err = parse_hotkey_keys(&json!({"keys":["CTRL","L"]})).expect("must parse");
    assert_eq!(err, vec!["ctrl".to_string(), "L".to_string()]);
}

#[test]
fn parse_hotkey_keys_rejects_empty_array() {
    let err = parse_hotkey_keys(&json!({"keys": []})).expect_err("must fail");
    assert!(err.contains("keys is required"));
}

#[test]
fn parse_hotkey_keys_rejects_unsupported_key_name() {
    let err = parse_hotkey_keys(&json!({"keys": ["CTRL", "Hyper"]})).expect_err("must fail");
    assert!(err.contains("unsupported hotkey key"));
}

#[cfg(target_os = "linux")]
#[test]
fn finalize_restore_result_keeps_typing_success() {
    let result =
        finalize_restore_result(Ok(()), Err("ibus failed".to_string())).expect("must pass");
    assert_eq!(
        result,
        Some("restore input method failed: ibus failed".to_string())
    );
}

#[cfg(target_os = "linux")]
#[test]
fn finalize_restore_result_combines_typing_and_restore_errors() {
    let err = finalize_restore_result(
        Err("typing failed".to_string()),
        Err("ibus failed".to_string()),
    )
    .expect_err("must fail");
    assert!(err.contains("typing failed"));
    assert!(err.contains("restore input method failed: ibus failed"));
}

#[cfg(target_os = "linux")]
#[test]
fn parse_active_window_id_output_trims_spaces() {
    let window_id = parse_active_window_id_output(b" 12345 \n").expect("must parse");
    assert_eq!(window_id, "12345");
}

#[cfg(target_os = "linux")]
#[test]
fn parse_active_window_id_output_rejects_empty_stdout() {
    let err = parse_active_window_id_output(b"  \n").expect_err("must fail");
    assert!(err.contains("empty output"));
}

#[cfg(target_os = "linux")]
#[test]
fn tokenize_text_for_xdotool_handles_mixed_language() {
    let got = tokenize_text_for_xdotool("我是 Ghost-OS 的 AI 助手。");
    assert_eq!(
        got,
        vec![
            XdotoolTextToken::Keysym("U6211".to_string()),
            XdotoolTextToken::Keysym("U662F".to_string()),
            XdotoolTextToken::AsciiChunk(" Ghost-OS ".to_string()),
            XdotoolTextToken::Keysym("U7684".to_string()),
            XdotoolTextToken::AsciiChunk(" AI ".to_string()),
            XdotoolTextToken::Keysym("U52A9".to_string()),
            XdotoolTextToken::Keysym("U624B".to_string()),
            XdotoolTextToken::Keysym("U3002".to_string()),
        ]
    );
}
