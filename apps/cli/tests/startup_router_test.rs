use std::fs;
use std::path::PathBuf;
use std::process::{Command, Output};
use std::time::{SystemTime, UNIX_EPOCH};

fn base_command() -> Command {
    let mut cmd = Command::new(env!("CARGO_BIN_EXE_ghost-cli"));
    cmd.env_remove("GHOST_CLI_PROFILE_STARTUP");
    cmd.env_remove("GHOST_CLI_PROFILE_STARTUP_FILE");
    cmd
}

fn stderr_text(output: &Output) -> String {
    String::from_utf8_lossy(&output.stderr).to_string()
}

#[test]
fn version_shortcuts_exit_success() {
    for flag in ["--version", "-v", "-V"] {
        let output = base_command()
            .arg(flag)
            .output()
            .expect("version command should run");
        assert!(
            output.status.success(),
            "{flag} should exit success, stderr={} ",
            stderr_text(&output)
        );

        let stdout = String::from_utf8_lossy(&output.stdout);
        assert!(
            stdout.contains("ghost-cli"),
            "expected version output, got {stdout:?}"
        );
    }
}

#[test]
fn startup_profiler_writes_report_when_enabled() {
    let profile_path = temp_profile_path();

    let output = base_command()
        .arg("--version")
        .env("GHOST_CLI_PROFILE_STARTUP", "1")
        .env("GHOST_CLI_PROFILE_STARTUP_FILE", &profile_path)
        .output()
        .expect("command should run");

    assert!(
        output.status.success(),
        "expected success, stderr={} ",
        stderr_text(&output)
    );

    let report = fs::read_to_string(&profile_path).expect("profile file should exist");
    assert!(
        report.contains("ghost-cli startup profile"),
        "missing profile header: {report:?}"
    );
    assert!(
        report.contains("entry"),
        "missing entry checkpoint: {report:?}"
    );
    assert!(
        report.contains("total"),
        "missing total duration: {report:?}"
    );

    let _ = fs::remove_file(profile_path);
}

#[test]
fn startup_profiler_rejects_invalid_flag_value() {
    let output = base_command()
        .arg("--version")
        .env("GHOST_CLI_PROFILE_STARTUP", "2")
        .output()
        .expect("command should run");

    assert!(
        !output.status.success(),
        "invalid profiler flag should fail"
    );

    let stderr = stderr_text(&output);
    assert!(
        stderr.contains("GHOST_CLI_PROFILE_STARTUP must be 1"),
        "unexpected stderr: {stderr:?}"
    );
}

#[test]
fn message_mode_rejects_blank_payload() {
    let output = base_command()
        .arg("--message")
        .arg("   ")
        .output()
        .expect("command should run");

    assert!(!output.status.success(), "blank --message should fail");

    let stderr = stderr_text(&output);
    assert!(
        stderr.contains("--message cannot be empty"),
        "unexpected stderr: {stderr:?}"
    );
}

fn temp_profile_path() -> PathBuf {
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("clock should be after unix epoch")
        .as_nanos();
    std::env::temp_dir().join(format!(
        "ghost-cli-startup-profile-{}-{stamp}.log",
        std::process::id()
    ))
}
