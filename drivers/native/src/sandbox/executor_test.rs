use super::{PythonSandbox, SandboxConfig};
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

fn make_repo_temp_dir() -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let dir = std::env::current_dir()
        .expect("resolve current dir")
        .join(format!(
            "ghost_os_sandbox_executor_test_{}_{}",
            std::process::id(),
            nanos
        ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}

fn sandbox_for(root: &Path) -> PythonSandbox {
    let root = root.to_string_lossy().to_string();
    let config = SandboxConfig {
        allowed_read_paths: vec![root.clone()],
        allowed_write_paths: vec![root],
        ..SandboxConfig::default()
    };
    PythonSandbox::new(config)
}

fn escape_python_path(path: &Path) -> String {
    path.to_string_lossy()
        .replace('\\', "\\\\")
        .replace('\'', "\\'")
}

#[test]
fn test_basic_script_execution() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let result = sandbox.execute_blocking("print('Hello from sandbox')");

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert_eq!(result.output.trim(), "Hello from sandbox");
}

#[test]
fn test_pathlib_module_restriction() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking("import pathlib\nprint(pathlib.Path('.'))");

    assert!(result.error.is_some(), "expected module restriction error");
    let module_error = result.error.clone().unwrap_or_default();
    assert!(
        module_error.contains("not allowed"),
        "unexpected module restriction error: {:?}",
        result.error
    );
}

#[test]
fn test_subprocess_shim_check_output() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking(
        "import subprocess\nprint(subprocess.check_output(['printf', 'shim-ok'], text=True))",
    );

    assert!(
        result.error.is_none(),
        "unexpected subprocess shim error: {:?}",
        result.error
    );
    assert!(result.output.contains("shim-ok"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "bash_exec");
}

#[test]
fn test_os_is_preloaded_and_popen_works() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking("print(os.popen(\"printf 'os-ok'\").read())");

    assert!(
        result.error.is_none(),
        "unexpected os.popen shim error: {:?}",
        result.error
    );
    assert!(result.output.contains("os-ok"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "bash_exec");
}

#[test]
fn test_json_module_is_allowed() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking("import json\nprint(json.dumps({'ok': True}))");

    assert!(
        result.error.is_none(),
        "unexpected json import error: {:?}",
        result.error
    );
    assert!(result.output.contains("{\"ok\": true}"));
}

#[test]
fn test_tool_call_bash_exec() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let script = r#"
result = bash_exec(command='printf "test"')
print(f"Result: {result}")
"#;
    let result = sandbox.execute_blocking(script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("Result: test"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "bash_exec");
}

#[test]
fn test_tool_call_bash_exec_with_positional_arg() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let result = sandbox.execute_blocking("print(bash_exec('printf \"positional\"'))");

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("positional"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "bash_exec");
}

#[test]
fn test_tool_call_list_files() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let script = r#"
files = list_files(path='.')
print(f"Found {len(files)} files")
"#;
    let result = sandbox.execute_blocking(script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("Found"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "list_files");
}

#[test]
fn test_tool_call_list_files_with_positional_arg() {
    let root = make_repo_temp_dir();
    fs::write(root.join("alpha.txt"), "alpha").expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "files = list_files('{path}')\nprint(files[0])",
        path = escape_python_path(&root)
    );
    let result = sandbox.execute_blocking(&script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert!(result.output.contains("alpha.txt"));
    assert_eq!(result.tool_calls_log.len(), 1);
    assert_eq!(result.tool_calls_log[0].tool, "list_files");

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_tools_object_is_not_defined() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());
    let result = sandbox.execute_blocking("tools.list_files(path='.')");

    assert!(result.error.is_some(), "expected name error");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("name 'tools' is not defined"),
        "unexpected error message: {err}"
    );
}

#[test]
fn test_open_shim_reads_text_file() {
    let root = make_repo_temp_dir();
    let file = root.join("sample.txt");
    fs::write(&file, "alpha\nbeta\n").expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "with open('{path}', 'r') as handle:\n    print(handle.readline().strip())\n    print(handle.read().strip())",
        path = escape_python_path(&file)
    );
    let result = sandbox.execute_blocking(&script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert_eq!(result.output.trim(), "alpha\nbeta");

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_open_shim_writes_and_appends_text_file() {
    let root = make_repo_temp_dir();
    let file = root.join("written.txt");

    let sandbox = sandbox_for(&root);
    let script = format!(
        "with open('{path}', 'w') as handle:\n    handle.write('alpha')\nwith open('{path}', 'a') as handle:\n    handle.write('\\nbeta')\nwith open('{path}', 'r') as handle:\n    print(handle.read())",
        path = escape_python_path(&file)
    );
    let result = sandbox.execute_blocking(&script);

    assert!(
        result.error.is_none(),
        "unexpected error: {:?}",
        result.error
    );
    assert_eq!(result.output.trim(), "alpha\nbeta");

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_open_shim_rejects_binary_mode() {
    let root = make_repo_temp_dir();
    let file = root.join("binary.txt");
    fs::write(&file, "alpha").expect("write fixture");

    let sandbox = sandbox_for(&root);
    let script = format!("open('{path}', 'rb')", path = escape_python_path(&file));
    let result = sandbox.execute_blocking(&script);

    assert!(result.error.is_some(), "expected binary mode failure");
    let err = result.error.unwrap_or_default();
    assert!(
        err.contains("does not support binary mode"),
        "unexpected error message: {err}"
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_dangerous_pattern_detection() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let result = sandbox.execute_blocking("eval('print(1)')");

    assert!(result.error.is_some(), "expected dangerous pattern error");
    let safety_error = result.error.clone().unwrap_or_default();
    assert!(
        safety_error.contains("dangerous pattern"),
        "unexpected dangerous pattern error: {:?}",
        result.error
    );
}
