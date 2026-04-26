use serde_json::json;
use std::{
    fs,
    path::PathBuf,
    time::{SystemTime, UNIX_EPOCH},
};

use super::dispatch_action;
use super::handlers::{handle_apply_diff, handle_export_file, handle_list_files, handle_read_file};

#[test]
fn handle_read_file_returns_numbered_content() {
    let root = make_temp_dir();
    let file = root.join("sample.txt");
    fs::write(&file, "alpha\nbeta\ngamma\n").expect("write fixture");

    let response = handle_read_file(&json!({
        "path": file.to_string_lossy(),
        "start_line": 2,
        "end_line": 3
    }));

    assert_eq!(response.status, "success");
    assert_eq!(response.payload["requested_start_line"], 2);
    assert_eq!(response.payload["returned_start_line"], 2);
    let content = response.payload["content"]
        .as_str()
        .expect("content should be a string");
    assert!(content.contains("2 | beta"));
    assert!(content.contains("3 | gamma"));

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_list_files_returns_sorted_entries_with_compatible_fields() {
    let root = make_temp_dir();
    fs::create_dir_all(root.join("docs")).expect("create docs");
    fs::write(root.join("z-last.txt"), "z").expect("write z-last.txt");
    fs::write(root.join("a-first.txt"), "a").expect("write a-first.txt");

    let response = handle_list_files(&json!({
        "path": root.to_string_lossy(),
    }));

    assert_eq!(response.status, "success");
    assert_eq!(response.payload["path"], root.to_string_lossy().to_string());
    assert_eq!(
        response.payload["entries"],
        json!(["a-first.txt", "docs/", "z-last.txt"])
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_list_files_blocks_sensitive_directory_names() {
    let root = make_temp_dir();
    let blocked = root.join("credentials-vault");
    fs::create_dir_all(&blocked).expect("create blocked dir");

    let response = handle_list_files(&json!({
        "path": blocked.to_string_lossy(),
    }));

    assert_eq!(response.status, "error");
    assert!(response.error.contains("sensitive file blocked"));

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_apply_diff_updates_file() {
    let root = make_temp_dir();
    let file = root.join("patch.txt");
    fs::write(&file, "alpha\nbeta\ngamma\n").expect("write fixture");

    let response = handle_apply_diff(&json!({
        "path": file.to_string_lossy(),
        "diff_text": "@@ -1,3 +1,3 @@\n alpha\n-beta\n+beta2\n gamma\n"
    }));

    assert_eq!(response.status, "success");
    let updated = fs::read_to_string(&file).expect("read updated file");
    assert_eq!(updated, "alpha\nbeta2\ngamma\n");
    assert_eq!(response.payload["hunk_count"], 1);

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_export_file_copies_file_into_session_artifact_directory() {
    let root = make_temp_dir();
    let source = root.join("notes.txt");
    let artifact_root = root.join("artifacts");
    fs::write(&source, "hello export\n").expect("write source file");

    let response = handle_export_file(&json!({
        "path": source.to_string_lossy(),
        "session_id": "session-1",
        "artifact_root": artifact_root.to_string_lossy(),
        "artifact_id": "artifact-1",
        "max_bytes": 1024
    }));

    assert_eq!(response.status, "success");
    assert_eq!(response.payload["artifact_id"], "artifact-1");
    assert_eq!(response.payload["filename"], "notes.txt");
    let stored_path = response.payload["stored_path"]
        .as_str()
        .expect("stored_path should be a string");
    assert!(stored_path.contains("artifacts/sessions/session-1/artifact-1.txt"));
    assert!(PathBuf::from(stored_path).exists());

    fs::remove_dir_all(root).ok();
}

#[test]
fn dispatch_action_returns_none_for_unknown_file_action() {
    assert!(dispatch_action("PING", &json!({})).is_none());
}

fn make_temp_dir() -> PathBuf {
    let mut dir = std::env::current_dir().expect("resolve current dir");
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    dir.push(format!(
        "ghost-os-native-file-actions-test-{}-{nanos}",
        std::process::id()
    ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}
