use serde_json::Value;

use crate::sandbox::SandboxConfig;

use super::types::ScriptExecutionBudget;

pub(super) fn resolve_script_budget(
    params: &Value,
    config: &SandboxConfig,
) -> Result<ScriptExecutionBudget, String> {
    let timeout_ms = clamp_budget(
        optional_u64(params, "timeout_ms")?,
        config.default_script_timeout_ms,
        config.max_script_timeout_ms,
    );
    let max_memory_mb = clamp_budget(
        optional_u64(params, "max_memory_mb")?,
        config.default_script_memory_mb,
        config.max_script_memory_mb,
    );

    Ok(ScriptExecutionBudget {
        timeout_ms,
        max_memory_mb,
    })
}

fn optional_u64(params: &Value, field: &str) -> Result<Option<u64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };

    raw.as_u64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a non-negative integer"))
}

fn clamp_budget(requested: Option<u64>, default_value: u64, max_value: u64) -> u64 {
    let value = requested.unwrap_or(default_value);
    resolve_budget_value(value, default_value).min(max_value)
}

fn resolve_budget_value(requested: u64, default_value: u64) -> u64 {
    if requested == 0 {
        return default_value;
    }
    requested
}

#[cfg(test)]
mod tests {
    use super::{clamp_budget, optional_u64, resolve_script_budget};
    use crate::sandbox::SandboxConfig;
    use serde_json::json;

    #[test]
    fn resolve_script_budget_uses_defaults_and_caps() {
        let config = SandboxConfig::default();
        let budget = resolve_script_budget(
            &json!({
                "timeout_ms": 120_000,
                "max_memory_mb": 2_048
            }),
            &config,
        )
        .expect("budget should resolve");

        assert_eq!(budget.timeout_ms, config.max_script_timeout_ms);
        assert_eq!(budget.max_memory_mb, config.max_script_memory_mb);
    }

    #[test]
    fn resolve_script_budget_treats_zero_as_default() {
        let config = SandboxConfig::default();
        let budget = resolve_script_budget(
            &json!({
                "timeout_ms": 0,
                "max_memory_mb": 0
            }),
            &config,
        )
        .expect("budget should resolve");

        assert_eq!(budget.timeout_ms, config.default_script_timeout_ms);
        assert_eq!(budget.max_memory_mb, config.default_script_memory_mb);
    }

    #[test]
    fn optional_u64_rejects_invalid_types() {
        let err = optional_u64(&json!({"timeout_ms": "slow"}), "timeout_ms")
            .expect_err("string value should fail");
        assert_eq!(err, "timeout_ms must be a non-negative integer");
    }

    #[test]
    fn clamp_budget_returns_default_for_none() {
        assert_eq!(clamp_budget(None, 10, 20), 10);
    }
}
