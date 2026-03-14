use serde_json::{Value, json};

use super::manager::CodexCommandManager;
use super::start::handle_start;
use super::status::handle_status;

#[test]
fn test_start_requires_prompt() {
    let mut manager = CodexCommandManager::new();
    let response = handle_start(&json!({"op": "start"}), &mut manager);
    assert_eq!(response.status, "error");
}

#[test]
fn test_start_rejects_disallowed_cwd() {
    let mut manager = CodexCommandManager::new();
    let response = handle_start(
        &json!({"op": "start", "prompt": "hi", "cwd": "/ghost-os/does-not-exist"}),
        &mut manager,
    );
    assert_eq!(response.status, "error");
}

#[test]
fn test_status_unknown_command_id_returns_error_status() {
    let mut manager = CodexCommandManager::new();
    let response = handle_status(&json!({"command_id": "missing"}), &mut manager);
    assert_eq!(response.status, "success");
    let status = response
        .payload
        .get("status")
        .and_then(Value::as_str)
        .unwrap_or("");
    assert_eq!(status, "error");
}
