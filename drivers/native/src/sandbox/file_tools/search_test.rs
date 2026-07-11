use super::search::{DEFAULT_SEARCH_MAX_RESULTS, SearchMatchOutput, search_files_impl};
use crate::sandbox::SandboxConfig;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

#[test]
fn test_search_files_rejects_empty_query() {
    let root = make_temp_dir("empty_query");
    let config = sandbox_config_for(&root);
    let err = search_files_impl(&config, "   ", &root.to_string_lossy(), 1)
        .expect_err("empty query should fail");
    assert_eq!(err, "query is required");
    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_returns_empty_for_no_match() {
    let root = make_temp_dir("no_match");
    fs::write(root.join("a.txt"), "alpha\nbeta\n").expect("write fixture");
    let config = sandbox_config_for(&root);
    let result = search_files_impl(
        &config,
        "missing-token",
        &root.to_string_lossy(),
        DEFAULT_SEARCH_MAX_RESULTS,
    )
    .expect("search files");
    assert!(result.is_empty());
    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_returns_stable_sorted_matches() {
    let root = make_temp_dir("stable_sort");
    fs::write(root.join("b.txt"), "zzz\nneedle beta\n").expect("write b");
    fs::write(root.join("a.txt"), "needle alpha\nx\nneedle zeta\n").expect("write a");
    let config = sandbox_config_for(&root);

    let result =
        search_files_impl(&config, "needle", &root.to_string_lossy(), 10).expect("search files");
    assert_eq!(
        to_sort_projection(&result),
        vec![
            ("a.txt".to_string(), 1, "needle alpha".to_string()),
            ("a.txt".to_string(), 3, "needle zeta".to_string()),
            ("b.txt".to_string(), 2, "needle beta".to_string()),
        ]
    );

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_honors_default_and_custom_max_results() {
    let root = make_temp_dir("max_results");
    let lines = (1..=60)
        .map(|line| format!("needle line {line}"))
        .collect::<Vec<_>>()
        .join("\n");
    fs::write(root.join("many.txt"), lines).expect("write fixture");
    let config = sandbox_config_for(&root);

    let default_result = search_files_impl(
        &config,
        "needle",
        &root.to_string_lossy(),
        DEFAULT_SEARCH_MAX_RESULTS,
    )
    .expect("search with default limit");
    assert_eq!(default_result.len(), DEFAULT_SEARCH_MAX_RESULTS);

    let custom_result =
        search_files_impl(&config, "needle", &root.to_string_lossy(), 3).expect("search custom");
    assert_eq!(custom_result.len(), 3);

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_excludes_tmp_directory() {
    let root = make_temp_dir("exclude_tmp");
    fs::create_dir_all(root.join("tmp")).expect("create tmp");
    fs::create_dir_all(root.join("src")).expect("create src");
    fs::write(root.join("tmp").join("audit.log"), "needle noisy").expect("write tmp");
    fs::write(root.join("src").join("main.txt"), "needle useful").expect("write src");
    let config = sandbox_config_for(&root);

    let result =
        search_files_impl(&config, "needle", &root.to_string_lossy(), 10).expect("search files");

    assert_eq!(result.len(), 1);
    assert!(result[0].path.ends_with("src/main.txt"));
    assert_eq!(result[0].text, "needle useful");

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_truncates_long_match_text() {
    let root = make_temp_dir("long_match");
    let long_line = format!("needle {}", "x".repeat(500));
    fs::write(root.join("long.txt"), &long_line).expect("write fixture");
    let config = sandbox_config_for(&root);

    let result =
        search_files_impl(&config, "needle", &root.to_string_lossy(), 10).expect("search files");

    assert_eq!(result.len(), 1);
    assert!(result[0].text.len() < long_line.len());
    assert!(result[0].text.ends_with("... [truncated]"));

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_search_files_rejects_invalid_path_inputs() {
    let root = make_temp_dir("invalid_path");
    let file_path = root.join("file.txt");
    fs::write(&file_path, "needle").expect("write fixture");
    let config = sandbox_config_for(&root);

    let non_dir_err = search_files_impl(&config, "needle", &file_path.to_string_lossy(), 1)
        .expect_err("file path should fail");
    assert!(non_dir_err.contains("path is not a directory"));

    let outside = make_temp_dir("outside_path");
    let forbidden_err = search_files_impl(&config, "needle", &outside.to_string_lossy(), 1)
        .expect_err("outside path should fail");
    assert!(forbidden_err.contains("allowed directories"));

    fs::remove_dir_all(outside).ok();
    fs::remove_dir_all(root).ok();
}

fn to_sort_projection(matches: &[SearchMatchOutput]) -> Vec<(String, usize, String)> {
    matches
        .iter()
        .map(|item| {
            let name = Path::new(&item.path)
                .file_name()
                .and_then(|value| value.to_str())
                .unwrap_or_default()
                .to_string();
            (name, item.line, item.text.clone())
        })
        .collect()
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
        "ghost_os_search_files_{label}_{}_{}",
        std::process::id(),
        nanos
    ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}
