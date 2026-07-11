use super::{prepend_path_dir, resolve_codex_launch_spec};
use std::ffi::OsString;
use std::fs;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[test]
fn resolve_codex_launch_spec_injects_node_directory_for_env_node_shebang() {
    let codex_path = temp_path("codex");
    let node_path = temp_path("node");
    fs::write(&codex_path, "#!/usr/bin/env node\nconsole.log('hi')\n")
        .expect("codex launcher should exist");
    fs::write(&node_path, "").expect("node binary should exist");
    make_executable(&codex_path);
    make_executable(&node_path);

    let spec = resolve_codex_launch_spec(codex_path.to_str(), node_path.to_str())
        .expect("launch spec should resolve");

    assert_eq!(spec.executable_path, codex_path);
    let path_override = spec.path_override.expect("path override should exist");
    let entries = std::env::split_paths(&path_override).collect::<Vec<_>>();
    assert_eq!(
        entries.first(),
        Some(&node_path.parent().expect("node parent").to_path_buf())
    );
}

#[test]
fn resolve_codex_launch_spec_reports_missing_node_for_env_launcher() {
    let codex_path = temp_path("codex-missing-node");
    fs::write(&codex_path, "#!/usr/bin/env node\nconsole.log('hi')\n")
        .expect("codex launcher should exist");
    make_executable(&codex_path);

    let err = resolve_codex_launch_spec(codex_path.to_str(), Some("/missing/node"))
        .expect_err("missing node should be explicit");

    assert!(err.contains("node_bin_path or GHOST_NODE_BIN_PATH"));
}

#[test]
fn resolve_codex_launch_spec_skips_path_override_for_non_node_launcher() {
    let codex_path = temp_path("codex-sh");
    fs::write(&codex_path, "#!/bin/sh\necho hi\n").expect("codex launcher should exist");
    make_executable(&codex_path);

    let spec = resolve_codex_launch_spec(codex_path.to_str(), None)
        .expect("shell launcher should resolve");

    assert!(spec.path_override.is_none());
}

#[test]
fn prepend_path_dir_puts_node_first_without_duplication() {
    let dir = Path::new("/node/bin");
    let current = join_paths(["/node/bin", "/usr/bin", "/opt/bin"]);

    let joined = prepend_path_dir(dir, Some(current)).expect("PATH should join");
    let entries = std::env::split_paths(&joined).collect::<Vec<_>>();

    assert_eq!(
        entries,
        vec![
            PathBuf::from("/node/bin"),
            PathBuf::from("/usr/bin"),
            PathBuf::from("/opt/bin"),
        ],
    );
}

fn temp_path(prefix: &str) -> PathBuf {
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system time should be valid")
        .as_nanos();
    let mut path = std::env::temp_dir();
    path.push(format!("ghost-native-codex-{prefix}-{stamp}"));
    path
}

#[cfg(unix)]
fn make_executable(path: &Path) {
    let mut perms = fs::metadata(path)
        .expect("metadata should exist")
        .permissions();
    perms.set_mode(0o755);
    fs::set_permissions(path, perms).expect("permissions should update");
}

#[cfg(not(unix))]
fn make_executable(_path: &Path) {}

fn join_paths<const N: usize>(paths: [&str; N]) -> OsString {
    std::env::join_paths(paths.iter().map(Path::new)).expect("path join should succeed")
}
