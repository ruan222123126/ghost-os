use serde_json::Value;

pub(crate) fn required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

pub(crate) fn required_text(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

pub(crate) fn optional_string(params: &Value, field: &str) -> Result<Option<String>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_str()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} must be a non-empty string"))?;
    Ok(Some(value))
}

pub(crate) fn optional_bool(params: &Value, field: &str) -> Result<Option<bool>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_bool()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a boolean"))
}

pub(crate) fn optional_usize(params: &Value, field: &str) -> Result<Option<usize>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_u64()
        .ok_or_else(|| format!("{field} must be a non-negative integer"))?;
    usize::try_from(value)
        .map(Some)
        .map_err(|_| format!("{field} is too large"))
}

pub(crate) fn optional_f64(params: &Value, field: &str) -> Result<Option<f64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_f64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a number"))
}

pub(crate) fn required_i32(params: &Value, field: &str) -> Result<i32, String> {
    let raw = params
        .get(field)
        .ok_or_else(|| format!("{field} is required"))?;
    let value = raw
        .as_i64()
        .ok_or_else(|| format!("{field} must be an integer"))?;
    i32::try_from(value).map_err(|_| format!("{field} is out of i32 range"))
}

pub(crate) fn optional_i32(params: &Value, field: &str) -> Result<Option<i32>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_i64()
        .ok_or_else(|| format!("{field} must be an integer"))?;
    i32::try_from(value)
        .map(Some)
        .map_err(|_| format!("{field} is out of i32 range"))
}

pub(crate) fn optional_display_id(params: &Value) -> Result<Option<u32>, String> {
    let Some(raw) = params.get("display_id") else {
        return Ok(None);
    };
    let value = raw
        .as_u64()
        .ok_or_else(|| "display_id must be a non-negative integer".to_string())?;
    u32::try_from(value)
        .map(Some)
        .map_err(|_| "display_id is too large".to_string())
}

#[cfg(test)]
mod tests {
    use super::{optional_bool, required_string, required_text};
    use serde_json::json;

    #[test]
    fn required_string_trims_text() {
        let value = required_string(&json!({"value":"  hi  "}), "value").expect("must parse");
        assert_eq!(value, "hi");
    }

    #[test]
    fn required_text_preserves_spaces() {
        let value = required_text(&json!({"value":"  hi  "}), "value").expect("must parse");
        assert_eq!(value, "  hi  ");
    }

    #[test]
    fn optional_bool_defaults_to_none() {
        assert_eq!(optional_bool(&json!({}), "flag").expect("must parse"), None);
    }
}
