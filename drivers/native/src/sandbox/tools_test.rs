use super::{
    PythonSandbox, SandboxConfig, file_tools::list_files_impl,
    tool_runtime::SCRIPT_SANDBOX_ALLOWED_TOOLS,
};
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

fn make_temp_dir() -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let dir = std::env::temp_dir().join(format!(
        "ghost_os_sandbox_tools_test_{}_{}",
        std::process::id(),
        nanos
    ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}

fn make_repo_temp_dir() -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let dir = std::env::current_dir()
        .expect("resolve current dir")
        .join(format!(
            "ghost_os_sandbox_tools_repo_test_{}_{}",
            std::process::id(),
            nanos
        ));
    fs::create_dir_all(&dir).expect("create repo temp dir");
    dir
}

fn sandbox_for(root: &Path) -> PythonSandbox {
    let mut config = SandboxConfig::default();
    let root = root.to_string_lossy().to_string();
    config.allowed_read_paths = vec![root.clone()];
    config.allowed_write_paths = vec![root];
    PythonSandbox::new(config)
}

fn escape_python_path(path: &Path) -> String {
    path.to_string_lossy()
        .replace('\\', "\\\\")
        .replace('\'', "\\'")
}

#[test]
fn test_list_files_relative_and_absolute_paths_match() {
    let cwd = std::env::current_dir().expect("resolve current dir");
    let root = make_repo_temp_dir();
    fs::create_dir_all(root.join("nested")).expect("create nested dir");
    fs::write(root.join("alpha.txt"), "alpha").expect("write alpha.txt");

    let config = SandboxConfig {
        allowed_read_paths: vec![root.to_string_lossy().to_string()],
        ..SandboxConfig::default()
    };

    let absolute = list_files_impl(&config, &root.to_string_lossy()).expect("list absolute path");
    let relative_path = root
        .strip_prefix(&cwd)
        .expect("strip cwd prefix")
        .to_string_lossy()
        .to_string();
    let relative = list_files_impl(&config, &relative_path).expect("list relative path");

    assert_eq!(absolute.entries, relative.entries);

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_list_files_returns_stable_sorted_entries() {
    let root = make_temp_dir();
    fs::create_dir_all(root.join("docs")).expect("create docs dir");
    fs::write(root.join("z-last.txt"), "z").expect("write z-last.txt");
    fs::write(root.join("a-first.txt"), "a").expect("write a-first.txt");

    let config = SandboxConfig {
        allowed_read_paths: vec![root.to_string_lossy().to_string()],
        ..SandboxConfig::default()
    };

    let result = list_files_impl(&config, &root.to_string_lossy()).expect("list files");
    assert_eq!(result.entries, vec!["a-first.txt", "docs/", "z-last.txt"]);

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_read_file_paginated_range() {
    let root = make_temp_dir();
    let file = root.join("notes.txt");
    let content = (1..=20)
        .map(|line| format!("line-{line}"))
        .collect::<Vec<_>>()
        .join("\n");
    fs::write(&file, content).expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "result = tools.read_file(path='{}', start_line=5, end_line=7)\nprint(result)",
        escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("line-5\nline-6\nline-7"));

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_read_file_enforces_line_limit() {
    let root = make_temp_dir();
    let file = root.join("large.txt");
    let content = (1..=300)
        .map(|line| format!("row-{line}"))
        .collect::<Vec<_>>()
        .join("\n");
    fs::write(&file, content).expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "tools.read_file(path='{}', start_line=1, end_line=250)",
        escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(result.error.is_some(), "expected line limit failure");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("max 200 lines"),
        "unexpected error message: {err}"
    );

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_write_file_write_and_append_modes() {
    let root = make_temp_dir();
    let file = root.join("out.txt");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "tools.write_file(path='{path}', content='hello')\n\
tools.write_file(path='{path}', content=' world', mode='append')\n\
print(tools.read_file(path='{path}'))",
        path = escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("hello world"));

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_write_file_blocks_sensitive_patterns() {
    let root = make_temp_dir();
    let file = root.join(".env");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "tools.write_file(path='{}', content='SECRET=1')",
        escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(result.error.is_some(), "expected blocked write failure");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("sensitive file blocked"),
        "unexpected error message: {err}"
    );

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_apply_diff_success() {
    let root = make_temp_dir();
    let file = root.join("patch.txt");
    fs::write(&file, "alpha\nbeta\ngamma\n").expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "tools.apply_diff(path='{path}', diff_text='''@@ -1,3 +1,3 @@\n alpha\n-beta\n+beta2\n gamma\n''')\n\
print(tools.read_file(path='{path}'))",
        path = escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("alpha\nbeta2\ngamma"));

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_apply_diff_mismatch_returns_error() {
    let root = make_temp_dir();
    let file = root.join("patch-mismatch.txt");
    fs::write(&file, "one\ntwo\nthree\n").expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "tools.apply_diff(path='{path}', diff_text='''@@ -1,3 +1,3 @@\n one\n-four\n+TWO\n three\n''')",
        path = escape_python_path(&file)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(result.error.is_some(), "expected diff mismatch failure");
    let err = result.error.unwrap_or_default();
    assert!(err.contains("mismatch"), "unexpected error message: {err}");

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_search_files_returns_grep_style_matches() {
    let root = make_temp_dir();
    let src = root.join("src");
    fs::create_dir_all(&src).expect("create src dir");

    fs::write(src.join("a.txt"), "TODO: first\nnone\nTODO: second\n").expect("write a.txt");
    fs::write(src.join("b.txt"), "todo: lowercase\n").expect("write b.txt");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "matches = tools.search_files(keyword='TODO', dir_path='{path}', case_sensitive=True)\nfor item in matches:\n    print(item)",
        path = escape_python_path(&root)
    );

    let result = sandbox.execute_blocking(&script);
    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("src/a.txt:1:TODO: first"));
    assert!(result.output.contains("src/a.txt:3:TODO: second"));
    assert!(!result.output.contains("lowercase"));

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_bash_exec_truncates_large_stdout() {
    let root = make_temp_dir();
    let sandbox = sandbox_for(&root);

    let script = "output = tools.bash_exec(command=\"printf 'a%.0s' {1..2205}\")\nprint(output)";
    let result = sandbox.execute_blocking(script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(
        result.output.contains("output truncated")
            || result
                .tool_calls_log
                .iter()
                .any(|log| log.result.contains("output truncated")),
        "expected truncation marker in output or logs"
    );

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_bash_exec_enforces_per_call_timeout_budget() {
    if cfg!(target_os = "windows") {
        return;
    }

    let root = make_temp_dir();
    let mut config = SandboxConfig::default();
    let root_str = root.to_string_lossy().to_string();
    config.allowed_read_paths = vec![root_str.clone()];
    config.allowed_write_paths = vec![root_str];
    config.default_shell_timeout_ms = 10;
    config.max_shell_timeout_ms = 10;
    let sandbox = PythonSandbox::new(config);

    let result = sandbox.execute_blocking("tools.bash_exec(command='sleep 1')");
    assert!(result.error.is_some(), "expected timeout failure");
    let err = result.error.unwrap_or_default();
    assert!(err.contains("timed out"), "unexpected error message: {err}");

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_tool_call_logs_use_shared_truncation_budget() {
    let root = make_temp_dir();
    let file = root.join("large.txt");
    fs::write(&file, "a".repeat(256)).expect("write fixture");

    let mut config = SandboxConfig::default();
    let root_str = root.to_string_lossy().to_string();
    config.allowed_read_paths = vec![root_str.clone()];
    config.allowed_write_paths = vec![root_str];
    config.max_tool_log_chars = 32;
    let sandbox = PythonSandbox::new(config);

    let script = format!("tools.read_file(path='{}')", escape_python_path(&file));
    let result = sandbox.execute_blocking(&script);
    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(
        result
            .tool_calls_log
            .iter()
            .any(|log| log.result.contains("log truncated")),
        "expected shared log truncation marker in tool log"
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_script_sandbox_allowed_tools_are_explicit() {
    assert_eq!(
        SCRIPT_SANDBOX_ALLOWED_TOOLS,
        [
            "bash_exec",
            "list_files",
            "read_file",
            "write_file",
            "apply_diff",
            "search_files",
            "fetch_webpage",
        ]
    );
}

#[tokio::test]
async fn test_read_file_blocks_path_outside_allowlist() {
    let root = make_temp_dir();
    let outside = std::env::temp_dir().join("ghost_os_outside.txt");
    fs::write(&outside, "outside").expect("write outside fixture");

    let sandbox = sandbox_for(&root);
    let script = format!("tools.read_file(path='{}')", escape_python_path(&outside));

    let result = sandbox.execute_blocking(&script);
    assert!(result.error.is_some(), "expected allowlist failure");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("allowed directories"),
        "unexpected error message: {err}"
    );

    fs::remove_file(outside).ok();
    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_fetch_webpage_rejects_non_https() {
    let root = make_temp_dir();
    let sandbox = sandbox_for(&root);

    let result = sandbox.execute_blocking("tools.fetch_webpage(url='http://example.com')");
    assert!(result.error.is_some(), "expected https-only rejection");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("only HTTPS URLs allowed"),
        "unexpected error message: {err}"
    );

    fs::remove_dir_all(root).ok();
}

#[tokio::test]
async fn test_fetch_webpage_rate_limit() {
    let root = make_temp_dir();
    let mut config = SandboxConfig::default();
    let root_str = root.to_string_lossy().to_string();
    config.allowed_read_paths = vec![root_str.clone()];
    config.allowed_write_paths = vec![root_str];
    config.max_web_requests = 1;
    let sandbox = PythonSandbox::new(config);

    let script = r#"
try:
	tools.fetch_webpage(url='https://localhost')
except Exception:
	pass

# Second call should trip the per-script request limit.
tools.fetch_webpage(url='https://localhost')
"#;

    let result = sandbox.execute_blocking(script);
    assert!(result.error.is_some(), "expected rate limit failure");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("rate limit"),
        "unexpected error message: {err}"
    );

    fs::remove_dir_all(root).ok();
}
