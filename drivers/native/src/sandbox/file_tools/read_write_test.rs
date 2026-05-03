use super::read_write::write_file_impl;
use crate::sandbox::SandboxConfig;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[test]
fn test_write_file_creates_missing_parent_directories() {
    let root = make_temp_dir("create_parent_dirs");
    let config = sandbox_config_for(&root);
    let file = root.join("nested/deeper/out.txt");

    let result = write_file_impl(&config, &file.to_string_lossy(), "hello", "write")
        .expect("write file should create parent directories");

    assert!(result.message.contains("wrote 5 bytes"));
    assert_eq!(
        fs::read_to_string(&file).expect("read nested file"),
        "hello"
    );

    fs::remove_dir_all(root).ok();
}

fn sandbox_config_for(root: &Path) -> SandboxConfig {
    let mut config = SandboxConfig::default();
    let root = root.to_string_lossy().to_string();
    config.allowed_read_paths = vec![root.clone()];
    config.allowed_write_paths = vec![root];
    config
}

fn make_temp_dir(label: &str) -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let dir = std::env::temp_dir().join(format!(
        "ghost_os_read_write_{label}_{}_{}",
        std::process::id(),
        nanos
    ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}
