use ignore::WalkBuilder;
use regex::{Regex, RegexBuilder};
use std::fs;
use std::path::{Path, PathBuf};

use crate::sandbox::SandboxConfig;
use crate::sandbox::path_policy::{is_blocked_path, resolve_read_path};

pub(crate) struct SearchFilesOutput {
    pub(crate) dir_path: std::path::PathBuf,
    pub(crate) matches: Vec<String>,
}

pub(crate) fn search_files_impl(
    config: &SandboxConfig,
    keyword: &str,
    dir_path: &str,
    case_sensitive: bool,
) -> Result<SearchFilesOutput, String> {
    let keyword = keyword.trim();
    if keyword.is_empty() {
        return Err("keyword is required".to_string());
    }

    let canonical_dir = resolve_read_path(dir_path, config)?;
    if !canonical_dir.is_dir() {
        return Err(format!(
            "search path is not a directory: {}",
            canonical_dir.display()
        ));
    }

    let matcher = RegexBuilder::new(keyword)
        .case_insensitive(!case_sensitive)
        .build()
        .map_err(|err| format!("invalid regex pattern: {err}"))?;

    let mut walk_builder = WalkBuilder::new(&canonical_dir);
    walk_builder.standard_filters(true);
    walk_builder.max_depth(Some(config.max_search_depth));

    let mut files_scanned = 0usize;
    let mut matches = Vec::new();

    'walk: for entry in walk_builder.build() {
        let Some(path) = eligible_search_path(entry, config, &mut files_scanned) else {
            continue;
        };
        if files_scanned > config.max_search_files {
            break;
        }

        let content = match fs::read(&path) {
            Ok(content) => content,
            Err(_) => continue,
        };
        let display_path = relative_display_path(&path, &canonical_dir);
        if push_file_matches(
            &content,
            &matcher,
            &display_path,
            &mut matches,
            config.max_search_matches,
        ) {
            break 'walk;
        }
    }

    Ok(SearchFilesOutput {
        dir_path: canonical_dir,
        matches,
    })
}

fn eligible_search_path(
    entry: Result<ignore::DirEntry, ignore::Error>,
    config: &SandboxConfig,
    files_scanned: &mut usize,
) -> Option<PathBuf> {
    let entry = entry.ok()?;
    let file_type = entry.file_type()?;
    if !file_type.is_file() {
        return None;
    }

    *files_scanned += 1;
    let path = entry.into_path();
    if is_blocked_path(&path, &config.blocked_patterns) {
        return None;
    }
    Some(path)
}

fn relative_display_path(path: &Path, canonical_dir: &Path) -> String {
    path.strip_prefix(canonical_dir)
        .unwrap_or(path)
        .to_string_lossy()
        .to_string()
}

fn push_file_matches(
    content: &[u8],
    matcher: &Regex,
    display_path: &str,
    matches: &mut Vec<String>,
    max_matches: usize,
) -> bool {
    if content.contains(&0) {
        return false;
    }

    let text = String::from_utf8_lossy(content);
    for (line_number, line) in text.lines().enumerate() {
        if matcher.is_match(line) {
            matches.push(format!("{}:{}:{}", display_path, line_number + 1, line));
            if matches.len() >= max_matches {
                return true;
            }
        }
    }
    false
}
