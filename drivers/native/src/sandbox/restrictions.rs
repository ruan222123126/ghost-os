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

def _ghost_restricted_import(name, globals=None, locals=None, fromlist=(), level=0):
    root = name.split(".", 1)[0]
    if root not in _GHOST_ALLOWED_MODULES:
        raise ImportError(f"Module '{{name}}' is not allowed in sandbox")
    return _GHOST_ORIGINAL_IMPORT(name, globals, locals, fromlist, level)

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
    if script.len() > 10 * 1024 {
        return Err("script too long (max 10KB)".to_string());
    }

    let dangerous_functions = [
        "eval",
        "exec",
        "compile",
        "__import__",
        "open",
        "input",
        "globals",
        "locals",
        "vars",
        "dir",
    ];

    for function_name in &dangerous_functions {
        if contains_forbidden_call(script, function_name) {
            return Err(format!("dangerous pattern detected: {}(", function_name));
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
