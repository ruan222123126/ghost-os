use super::{
    codex_dir_from_node_bin_path, node_bin_path_from_value, nvm_version_bin_dirs,
    resolve_codex_executable, resolve_node_executable,
};
use crate::codex_cli::path_env::path_dirs;
use std::ffi::OsString;
use std::fs;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[test]
fn resolve_codex_executable_prefers_explicit_runnable_path() {
    let codex_path = temp_executable_path("explicit");
    fs::write(&codex_path, "#!/bin/sh\n").expect("codex file should exist");
    make_executable(&codex_path);

    let resolved =
        resolve_codex_executable(codex_path.to_str()).expect("explicit path should resolve");

    assert_eq!(resolved, codex_path);
}

#[test]
fn resolve_codex_executable_reports_invalid_explicit_path() {
    let path = temp_executable_path("missing");

    let err = resolve_codex_executable(path.to_str())
        .expect_err("missing explicit path should be explicit");

    assert!(err.contains("codex_cli_path or GHOST_CODEX_CLI_PATH"));
}

#[test]
fn resolve_node_executable_prefers_explicit_runnable_path() {
    let node_path = temp_executable_path("explicit-node");
    fs::write(&node_path, "").expect("node file should exist");
    make_executable(&node_path);

    let resolved =
        resolve_node_executable(node_path.to_str()).expect("explicit node path should resolve");

    assert_eq!(resolved, node_path);
}

#[test]
fn node_bin_path_from_value_uses_runnable_node_path() {
    let node_path = temp_executable_path("env-node");
    fs::write(&node_path, "").expect("node file should exist");
    make_executable(&node_path);

    let resolved = node_bin_path_from_value(Some(node_path.as_os_str().to_os_string()))
        .expect("NODE_BIN_PATH value should resolve");

    assert_eq!(resolved, node_path);
}

#[test]
fn codex_dir_from_node_bin_path_uses_node_parent_directory() {
    let node_path = Path::new("/tmp/tooling/node");
    let resolved = codex_dir_from_node_bin_path(node_path).expect("node parent should exist");

    assert_eq!(resolved, Path::new("/tmp/tooling"));
}

#[test]
fn nvm_version_bin_dirs_returns_matching_bin_directories() {
    let home = temp_directory_path("nvm-home");
    let node_bin = home.join(".nvm/versions/node/v24.14.0/bin");
    fs::create_dir_all(&node_bin).expect("nvm bin dir should exist");

    let dirs = nvm_version_bin_dirs(&home);

    assert!(dirs.contains(&node_bin));
}

#[test]
fn path_dirs_splits_search_path() {
    let joined = env_join(["/first/bin", "/second/bin"]);

    let dirs = path_dirs(Some(joined));

    assert_eq!(
        dirs,
        vec![PathBuf::from("/first/bin"), PathBuf::from("/second/bin")],
    );
}

fn temp_executable_path(prefix: &str) -> std::path::PathBuf {
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system time should be valid")
        .as_nanos();
    let mut path = std::env::temp_dir();
    path.push(format!("ghost-native-codex-{prefix}-{stamp}"));
    path
}

fn temp_directory_path(prefix: &str) -> std::path::PathBuf {
    let dir = temp_executable_path(prefix);
    fs::create_dir_all(&dir).expect("temp dir should exist");
    dir
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

fn env_join<const N: usize>(parts: [&str; N]) -> OsString {
    std::env::join_paths(parts.iter().map(Path::new)).expect("path join should succeed")
}
