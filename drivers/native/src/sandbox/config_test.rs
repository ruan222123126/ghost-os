use super::{merge_path_lists, normalize_existing_dirs, parse_path_list_csv};
use std::fs;
use std::path::PathBuf;
use std::time::{SystemTime, UNIX_EPOCH};

fn make_temp_dir(prefix: &str) -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let dir = std::env::temp_dir().join(format!(
        "ghost_os_sandbox_{prefix}_{}_{}",
        std::process::id(),
        nanos
    ));
    fs::create_dir_all(&dir).expect("create temp dir");
    dir
}

#[test]
fn test_normalize_existing_dirs_canonicalizes_and_deduplicates() {
    let root = make_temp_dir("normalize");
    let nested = root.join("nested");
    fs::create_dir_all(&nested).expect("create nested dir");

    let paths = vec![
        root.clone(),
        nested.join(".."),
        PathBuf::from("/missing/ghost-os"),
    ];
    let normalized = normalize_existing_dirs(paths);

    assert_eq!(normalized.len(), 1);
    assert_eq!(normalized[0], root.to_string_lossy());

    fs::remove_dir_all(root).ok();
}

#[test]
fn test_parse_path_list_csv_trims_and_discards_empty_items() {
    let parsed = parse_path_list_csv(" /tmp/a, ,/tmp/b ,, /tmp/c ");
    assert_eq!(
        parsed,
        vec![
            PathBuf::from("/tmp/a"),
            PathBuf::from("/tmp/b"),
            PathBuf::from("/tmp/c")
        ]
    );
}

#[test]
fn test_merge_path_lists_deduplicates_and_preserves_order() {
    let merged = merge_path_lists(
        vec!["/a".to_string(), "/b".to_string()],
        vec!["/b".to_string(), "/c".to_string(), "/a".to_string()],
    );
    assert_eq!(
        merged,
        vec!["/a".to_string(), "/b".to_string(), "/c".to_string()]
    );
}

#[test]
fn test_merge_path_lists_returns_empty_when_base_and_extra_are_empty() {
    let merged = merge_path_lists(Vec::new(), Vec::new());
    assert!(merged.is_empty());
}
