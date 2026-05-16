use std::collections::HashSet;
use std::env;
use std::ffi::OsString;
use std::path::PathBuf;

pub(crate) fn path_dirs(value: Option<OsString>) -> Vec<PathBuf> {
    let Some(value) = value else {
        return Vec::new();
    };
    env::split_paths(&value).collect()
}

pub(crate) fn home_dir() -> Option<PathBuf> {
    env_dir("HOME")
        .or_else(|| env_dir("USERPROFILE"))
        .or_else(home_dir_from_drive_path)
}

pub(crate) fn env_dir(name: &str) -> Option<PathBuf> {
    let value = env::var_os(name)?;
    let path = PathBuf::from(value);
    if path.as_os_str().is_empty() {
        return None;
    }
    Some(path)
}

pub(crate) fn dedupe_dirs(dirs: Vec<PathBuf>) -> Vec<PathBuf> {
    let mut seen = HashSet::new();
    let mut unique = Vec::new();
    for dir in dirs {
        let key = dir.to_string_lossy().to_string();
        if seen.insert(key) {
            unique.push(dir);
        }
    }
    unique
}

fn home_dir_from_drive_path() -> Option<PathBuf> {
    let drive = env::var_os("HOMEDRIVE")?;
    let path = env::var_os("HOMEPATH")?;
    let mut home = PathBuf::from(drive);
    home.push(path);
    Some(home)
}
