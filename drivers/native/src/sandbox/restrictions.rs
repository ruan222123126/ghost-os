// Restriction rules and validation helpers used by the native sandbox runtime.

use pyo3::prelude::*;
use pyo3::types::{PyDict, PyModule};

#[path = "restrictions_open_shim.rs"]
mod restrictions_open_shim;

use restrictions_open_shim::build_restriction_code;

const BLOCKED_BUILTINS: [&str; 4] = ["eval", "exec", "compile", "input"];

pub struct RestrictionGuard {
    builtins_module: Py<PyModule>,
    original_import: Py<PyAny>,
    original_builtins: Vec<(&'static str, Py<PyAny>)>,
}

impl RestrictionGuard {
    pub fn restore(self, py: Python<'_>) -> PyResult<()> {
        let builtins = self.builtins_module.bind(py);
        builtins.setattr("__import__", self.original_import)?;
        for (name, value) in self.original_builtins {
            builtins.setattr(name, value)?;
        }
        Ok(())
    }
}

pub fn setup_restricted_imports(
    py: Python<'_>,
    allowed_modules: &[String],
    scope: &Bound<'_, PyDict>,
) -> PyResult<RestrictionGuard> {
    let builtins = py.import_bound("builtins")?;
    let builtins_module = builtins.clone().unbind();
    let original_import = builtins.getattr("__import__")?.into_py(py);

    let original_builtins = capture_original_builtins(py, &builtins)?;
    let restriction_code = build_restriction_code(allowed_modules);
    py.run_bound(&restriction_code, Some(scope), Some(scope))?;

    Ok(RestrictionGuard {
        builtins_module,
        original_import,
        original_builtins,
    })
}

fn capture_original_builtins(
    py: Python<'_>,
    builtins: &Bound<'_, PyModule>,
) -> PyResult<Vec<(&'static str, Py<PyAny>)>> {
    let mut originals = Vec::with_capacity(BLOCKED_BUILTINS.len() + 1);
    for builtin_name in open_and_blocked_builtins() {
        let original = builtins.getattr(builtin_name)?.into_py(py);
        originals.push((builtin_name, original));
    }
    Ok(originals)
}

fn open_and_blocked_builtins() -> impl Iterator<Item = &'static str> {
    std::iter::once("open").chain(BLOCKED_BUILTINS)
}

pub fn validate_script_safety(script: &str) -> Result<(), String> {
    // 放宽长度限制到 50KB，适应更复杂的合法脚本。
    if script.len() > 50 * 1024 {
        return Err("script too long (max 50KB)".to_string());
    }

    // 运行时已通过 setup_restricted_imports 禁用危险内置函数，
    // 此处仅做函数调用级别的快速检查作为第一道防线，不作为唯一安全保障。
    let dangerous_calls = [
        "eval",
        "exec",
        "compile",
        "__import__",
        "input",
        "getattr", // 防止 getattr(builtins, 'eval') 绕过
        "setattr",
        "delattr",
    ];

    for call in &dangerous_calls {
        if contains_forbidden_call(script, call) {
            return Err(format!(
                "potentially dangerous pattern detected: {}",
                format_args!("{call}(")
            ));
        }
    }

    Ok(())
}

fn contains_forbidden_call(script: &str, function_name: &str) -> bool {
    let needle = format!("{function_name}(");
    for (idx, _) in script.match_indices(&needle) {
        let previous = script[..idx].chars().next_back();
        if previous
            .map(|ch| ch.is_ascii_alphanumeric() || ch == '_')
            .unwrap_or(false)
        {
            continue;
        }
        return true;
    }
    false
}
