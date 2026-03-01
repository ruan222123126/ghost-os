// Build script wiring for native driver dependencies and compile-time flags.

use std::collections::BTreeSet;
use std::env;
use std::path::Path;
use std::process::Command;

fn main() {
    println!("cargo:rerun-if-env-changed=PYO3_PYTHON");
    println!("cargo:rerun-if-env-changed=PYTHON_SYS_EXECUTABLE");

    let python = env::var("PYO3_PYTHON")
        .or_else(|_| env::var("PYTHON_SYS_EXECUTABLE"))
        .unwrap_or_else(|_| "python3".to_string());

    let output = Command::new(&python)
        .args([
            "-c",
            "import sysconfig; print(sysconfig.get_config_var('LIBDIR') or ''); print(sysconfig.get_config_var('LIBPL') or '')",
        ])
        .output();

    let output = match output {
        Ok(value) if value.status.success() => value,
        _ => return,
    };

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut paths = BTreeSet::new();
    for line in stdout.lines() {
        let path = line.trim();
        if !path.is_empty() && Path::new(path).exists() {
            paths.insert(path.to_string());
        }
    }

    for path in paths {
        println!("cargo:rustc-link-search=native={path}");
    }
}
