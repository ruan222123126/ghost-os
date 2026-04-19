use super::{PythonSandbox, SandboxConfig};

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
fn test_module_restriction() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let result = sandbox
        .execute_blocking("import subprocess\nprint(subprocess.check_output(['echo', 'x']))");

    assert!(result.error.is_some(), "expected module restriction error");
    let module_error = result.error.clone().unwrap_or_default();
    assert!(
        module_error.contains("not allowed"),
        "unexpected module restriction error: {:?}",
        result.error
    );
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
result = tools.bash_exec(command='printf "test"')
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
fn test_tool_call_list_files() {
    let sandbox = PythonSandbox::new(SandboxConfig::default());

    let script = r#"
files = tools.list_files(path='.')
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
