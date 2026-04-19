// Restriction rules and validation helpers used by the native sandbox runtime.

use pyo3::prelude::*;
use pyo3::types::PyModule;

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
) -> PyResult<RestrictionGuard> {
    let builtins = py.import_bound("builtins")?;
    let builtins_module = builtins.clone().unbind();
    let original_import = builtins.getattr("__import__")?.into_py(py);

    let mut original_builtins = Vec::new();
    for builtin_name in ["open", "eval", "exec", "compile", "input"] {
        let original = builtins.getattr(builtin_name)?.into_py(py);
        original_builtins.push((builtin_name, original));
    }

    let mut modules = allowed_modules.to_vec();
    modules.push("tools".to_string());
    let allowed_modules_json = serde_json::to_string(&modules).unwrap_or_else(|_| "[]".to_string());
    let restriction_code = format!(
        r#"
import builtins

_GHOST_ALLOWED_MODULES = set({allowed_modules_json})
_GHOST_ORIGINAL_IMPORT = builtins.__import__
_GHOST_IMPORT_DEPTH = 0
_GHOST_ORIGINAL_BUILTINS = {{
    "open": builtins.open,
    "eval": builtins.eval,
    "exec": builtins.exec,
    "compile": builtins.compile,
    "input": builtins.input,
}}

def _ghost_restricted_import(name, globals=None, locals=None, fromlist=(), level=0):
    global _GHOST_IMPORT_DEPTH
    root = name.split(".", 1)[0]
    allowed = root in _GHOST_ALLOWED_MODULES
    if not allowed and _GHOST_IMPORT_DEPTH > 0:
        allowed = True
    if not allowed and isinstance(globals, dict):
        importer_name = str(globals.get("__name__") or "")
        importer_package = str(globals.get("__package__") or "")
        importer_scope = importer_package or importer_name
        if importer_scope and importer_name != "__main__":
            importer_root = importer_scope.split(".", 1)[0]
            if importer_root in _GHOST_ALLOWED_MODULES:
                allowed = True
        elif level > 0 and importer_scope:
            importer_root = importer_scope.split(".", 1)[0]
            allowed = importer_root in _GHOST_ALLOWED_MODULES
    if not allowed:
        raise ImportError(f"Module '{{name}}' is not allowed in sandbox")

    previous_builtins = {{}}
    for _builtin_name, _original in _GHOST_ORIGINAL_BUILTINS.items():
        previous_builtins[_builtin_name] = getattr(builtins, _builtin_name)
        setattr(builtins, _builtin_name, _original)
    _GHOST_IMPORT_DEPTH += 1
    try:
        return _GHOST_ORIGINAL_IMPORT(name, globals, locals, fromlist, level)
    finally:
        _GHOST_IMPORT_DEPTH -= 1
        for _builtin_name, _previous in previous_builtins.items():
            setattr(builtins, _builtin_name, _previous)

def _ghost_blocked_builtin(*_args, **_kwargs):
    raise PermissionError("This builtin is not allowed in sandbox")

builtins.__import__ = _ghost_restricted_import
for _builtin_name in ("open", "eval", "exec", "compile", "input"):
    setattr(builtins, _builtin_name, _ghost_blocked_builtin)
"#
    );
    py.run_bound(&restriction_code, None, None)?;

    Ok(RestrictionGuard {
        builtins_module,
        original_import,
        original_builtins,
    })
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
        "open",
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
