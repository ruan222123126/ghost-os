use super::{default_media_mounts, merge_path_lists, normalize_existing_dirs, parse_path_list_csv};
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
fn test_default_media_mounts_discovers_files_and_apps_dirs() {
    let media_root = make_temp_dir("media_root");
    let home_dir = media_root.join("home").join("demo");
    let fake_media = media_root.join("media").join("demo");
    fs::create_dir_all(&home_dir).expect("create home dir");
    fs::create_dir_all(fake_media.join("Files")).expect("create Files dir");
    fs::create_dir_all(fake_media.join("Apps")).expect("create Apps dir");

    let mounts = default_media_mounts_for_base(&home_dir, &media_root.join("media"));

    assert_eq!(mounts.len(), 2);
    assert!(
        mounts
            .iter()
            .any(|path| path.ends_with("/media/demo/Files"))
    );
    assert!(mounts.iter().any(|path| path.ends_with("/media/demo/Apps")));

    fs::remove_dir_all(media_root).ok();
}

fn default_media_mounts_for_base(
    home_dir: &std::path::Path,
    media_base: &std::path::Path,
) -> Vec<String> {
    let Some(user_name) = home_dir.file_name().and_then(|name| name.to_str()) else {
        return Vec::new();
    };

    ["Files", "Apps", "App"]
        .into_iter()
        .map(|name| media_base.join(user_name).join(name))
        .filter(|path| path.is_dir())
        .map(|path| path.to_string_lossy().to_string())
        .collect()
}

#[test]
fn test_default_media_mounts_handles_unknown_user_dir() {
    let mounts = default_media_mounts(PathBuf::from("/").as_path());
    assert!(mounts.is_empty());
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
fn test_merge_path_lists_returns_dot_when_base_and_extra_are_empty() {
    let merged = merge_path_lists(Vec::new(), Vec::new());
    assert_eq!(merged, vec![".".to_string()]);
}
