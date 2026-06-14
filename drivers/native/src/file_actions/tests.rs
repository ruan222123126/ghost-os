use serde_json::json;
use std::{
    fs,
    path::PathBuf,
    time::{SystemTime, UNIX_EPOCH},
};

use super::dispatch_action;
use super::handlers::{
    handle_apply_diff, handle_list_files, handle_read_file, handle_search_files, handle_write_file,
};

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
fn handle_search_files_returns_sorted_matches() {
    let root = make_temp_dir();
    fs::write(root.join("b.txt"), "zzz\nneedle beta\n").expect("write b.txt");
    fs::write(root.join("a.txt"), "needle alpha\nx\nneedle zeta\n").expect("write a.txt");

    let response = handle_search_files(&json!({
        "query": "needle",
        "path": root.to_string_lossy(),
    }));

    assert_eq!(response.status, "success");
    assert_eq!(
        response.payload["matches"],
        json!([
            {"path": root.join("a.txt").to_string_lossy().to_string(), "line": 1, "text": "needle alpha"},
            {"path": root.join("a.txt").to_string_lossy().to_string(), "line": 3, "text": "needle zeta"},
            {"path": root.join("b.txt").to_string_lossy().to_string(), "line": 2, "text": "needle beta"},
        ])
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_search_files_rejects_missing_query() {
    let response = handle_search_files(&json!({"path": "."}));
    assert_eq!(response.status, "error");
    assert_eq!(response.error, "query is required");
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
fn handle_write_file_writes_and_appends_content() {
    let root = make_temp_dir();
    let file = root.join("write.txt");

    let write = handle_write_file(&json!({
        "path": file.to_string_lossy(),
        "content": "alpha"
    }));
    assert_eq!(write.status, "success");

    let append = handle_write_file(&json!({
        "path": file.to_string_lossy(),
        "content": "",
        "mode": "append"
    }));
    assert_eq!(append.status, "success");
    assert_eq!(
        fs::read_to_string(&file).expect("read written file"),
        "alpha"
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn handle_write_file_rejects_invalid_mode() {
    let root = make_temp_dir();
    let file = root.join("write.txt");

    let response = handle_write_file(&json!({
        "path": file.to_string_lossy(),
        "content": "alpha",
        "mode": ""
    }));

    assert_eq!(response.status, "error");
    assert_eq!(response.error, "mode must be 'write' or 'append'");

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
fn dispatch_action_returns_none_for_unknown_file_action() {
    assert!(dispatch_action("PING", &json!({})).is_none());
}

#[test]
fn dispatch_action_routes_search_and_write_file_actions() {
    let root = make_temp_dir();
    fs::write(root.join("search.txt"), "needle\n").expect("write search fixture");

    let search = dispatch_action(
        "SEARCH_FILES",
        &json!({"query": "needle", "path": root.to_string_lossy()}),
    )
    .expect("SEARCH_FILES should dispatch");
    assert_eq!(search.status, "success");

    let write = dispatch_action("WRITE_FILE", &json!({"path": ".env", "content": "y"}))
        .expect("WRITE_FILE should dispatch");
    assert_eq!(write.status, "error");

    fs::remove_dir_all(root).ok();
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
