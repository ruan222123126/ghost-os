use super::types::ScreenRegion;
use crate::json_params::optional_display_id;
use serde_json::{Map, Value};

#[derive(Clone, Copy, Debug)]
pub(crate) struct ScaleRange {
    pub(crate) min: f64,
    pub(crate) max: f64,
    pub(crate) step: f64,
}

pub(crate) fn parse_optional_display_id(params: &Value) -> Result<Option<u32>, String> {
    optional_display_id(params)
}

pub(crate) fn parse_capture_region(
    params: &Value,
    full_width: u32,
    full_height: u32,
) -> Result<ScreenRegion, String> {
    let default_region = full_region(full_width, full_height)?;
    let Some(raw) = params.get("region") else {
        return Ok(default_region);
    };
    parse_region_value(raw, full_width, full_height, true)
}

pub(crate) fn parse_required_region(
    params: &Value,
    field: &str,
    full_width: u32,
    full_height: u32,
) -> Result<ScreenRegion, String> {
    let raw = params
        .get(field)
        .ok_or_else(|| format!("{field} is required"))?;
    parse_region_value(raw, full_width, full_height, false)
}

pub(crate) fn parse_scale_range(params: &Value) -> Result<Option<ScaleRange>, String> {
    let Some(raw) = params.get("scale_range") else {
        return Ok(None);
    };
    let obj = raw
        .as_object()
        .ok_or_else(|| "scale_range must be an object".to_string())?;
    let min = obj
        .get("min")
        .and_then(Value::as_f64)
        .ok_or_else(|| "scale_range.min must be a number".to_string())?;
    let max = obj
        .get("max")
        .and_then(Value::as_f64)
        .ok_or_else(|| "scale_range.max must be a number".to_string())?;
    let step = match obj.get("step") {
        Some(value) => value
            .as_f64()
            .ok_or_else(|| "scale_range.step must be a number".to_string())?,
        None => 0.1,
    };
    if min <= 0.0 || max <= 0.0 {
        return Err("scale_range.min and scale_range.max must be greater than 0".to_string());
    }
    if min > max {
        return Err("scale_range.min must be <= scale_range.max".to_string());
    }
    if step <= 0.0 {
        return Err("scale_range.step must be greater than 0".to_string());
    }
    Ok(Some(ScaleRange { min, max, step }))
}

fn full_region(full_width: u32, full_height: u32) -> Result<ScreenRegion, String> {
    Ok(ScreenRegion {
        x: 0,
        y: 0,
        width: i32::try_from(full_width).map_err(|_| "screen width is too large".to_string())?,
        height: i32::try_from(full_height).map_err(|_| "screen height is too large".to_string())?,
    })
}

fn parse_region_value(
    raw: &Value,
    full_width: u32,
    full_height: u32,
    clamp_to_bounds: bool,
) -> Result<ScreenRegion, String> {
    let region_obj = raw
        .as_object()
        .ok_or_else(|| "region must be an object".to_string())?;
    let x = parse_region_field(region_obj, "x")?;
    let y = parse_region_field(region_obj, "y")?;
    let width = parse_region_field(region_obj, "width")?;
    let height = parse_region_field(region_obj, "height")?;

    if x < 0 || y < 0 {
        return Err("region.x and region.y must be non-negative".to_string());
    }
    if width <= 0 || height <= 0 {
        return Err("region.width and region.height must be greater than 0".to_string());
    }

    let full_width_i32 =
        i32::try_from(full_width).map_err(|_| "screen width is too large".to_string())?;
    let full_height_i32 =
        i32::try_from(full_height).map_err(|_| "screen height is too large".to_string())?;
    let region = ScreenRegion {
        x,
        y,
        width,
        height,
    };
    if clamp_to_bounds {
        return clamp_region_to_bounds(region, full_width_i32, full_height_i32);
    }
    if x + width > full_width_i32 || y + height > full_height_i32 {
        return Err(format!(
            "region {}x{} at ({x},{y}) exceeds screen bounds {}x{}",
            width, height, full_width, full_height
        ));
    }
    Ok(region)
}

fn clamp_region_to_bounds(
    region: ScreenRegion,
    full_width: i32,
    full_height: i32,
) -> Result<ScreenRegion, String> {
    if full_width <= 0 || full_height <= 0 {
        return Err(format!(
            "screen bounds must be positive, got {}x{}",
            full_width, full_height
        ));
    }
    let max_x = full_width - 1;
    let max_y = full_height - 1;
    let x = region.x.clamp(0, max_x);
    let y = region.y.clamp(0, max_y);
    let width = region.width.min(full_width - x).max(1);
    let height = region.height.min(full_height - y).max(1);
    Ok(ScreenRegion {
        x,
        y,
        width,
        height,
    })
}

fn parse_region_field(region: &Map<String, Value>, field: &str) -> Result<i32, String> {
    let raw = region
        .get(field)
        .ok_or_else(|| format!("region.{field} is required"))?;
    let value = raw
        .as_i64()
        .ok_or_else(|| format!("region.{field} must be an integer"))?;
    i32::try_from(value).map_err(|_| format!("region.{field} is out of i32 range"))
}

#[cfg(test)]
mod tests {
    use super::{parse_capture_region, parse_required_region, parse_scale_range};
    use serde_json::json;

    #[test]
    fn parse_capture_region_defaults_to_full_screen() {
        let region = parse_capture_region(&json!({}), 1920, 1080).expect("must parse");
        assert_eq!(region.width, 1920);
        assert_eq!(region.height, 1080);
    }

    #[test]
    fn parse_capture_region_clamps_to_screen_bounds() {
        let region = parse_capture_region(
            &json!({
                "region": {"x": 0, "y": 0, "width": 2200, "height": 1400}
            }),
            1920,
            1080,
        )
        .expect("must clamp");
        assert_eq!(region.x, 0);
        assert_eq!(region.y, 0);
        assert_eq!(region.width, 1920);
        assert_eq!(region.height, 1080);
    }

    #[test]
    fn parse_capture_region_clamps_when_origin_is_outside_bounds() {
        let region = parse_capture_region(
            &json!({
                "region": {"x": 2500, "y": 1200, "width": 100, "height": 100}
            }),
            1920,
            1080,
        )
        .expect("must clamp");
        assert_eq!(region.x, 1919);
        assert_eq!(region.y, 1079);
        assert_eq!(region.width, 1);
        assert_eq!(region.height, 1);
    }

    #[test]
    fn parse_required_region_stays_strict_for_out_of_bounds() {
        let err = parse_required_region(
            &json!({
                "region": {"x": 0, "y": 0, "width": 2200, "height": 1400}
            }),
            "region",
            1920,
            1080,
        )
        .expect_err("must fail");
        assert!(err.contains("exceeds screen bounds"));
    }

    #[test]
    fn parse_scale_range_rejects_invalid_min_max() {
        let err = parse_scale_range(&json!({
            "scale_range": {"min": 2.0, "max": 1.0}
        }))
        .expect_err("must fail");
        assert!(err.contains("min must be <="));
    }

    #[test]
    fn parse_required_region_rejects_negative_coordinates() {
        let err = parse_required_region(
            &json!({
                "region": {"x": -1, "y": 0, "width": 10, "height": 10}
            }),
            "region",
            100,
            100,
        )
        .expect_err("must fail");
        assert!(err.contains("must be non-negative"));
    }

    #[test]
    fn parse_required_region_rejects_non_integer_fields() {
        let err = parse_required_region(
            &json!({
                "region": {"x": 1.5, "y": 0, "width": 10, "height": 10}
            }),
            "region",
            100,
            100,
        )
        .expect_err("must fail");
        assert!(err.contains("must be an integer"));
    }

    #[test]
    fn parse_required_region_rejects_i32_overflow() {
        let err = parse_required_region(
            &json!({
                "region": {"x": 2147483648_i64, "y": 0, "width": 10, "height": 10}
            }),
            "region",
            100,
            100,
        )
        .expect_err("must fail");
        assert!(err.contains("out of i32 range"));
    }

    #[test]
    fn parse_scale_range_defaults_step_to_point_one() {
        let range = parse_scale_range(&json!({
            "scale_range": {"min": 0.8, "max": 1.2}
        }))
        .expect("must parse")
        .expect("must exist");
        assert_eq!(range.step, 0.1);
    }

    #[test]
    fn parse_scale_range_rejects_non_numeric_step() {
        let err = parse_scale_range(&json!({
            "scale_range": {"min": 0.8, "max": 1.2, "step": "bad"}
        }))
        .expect_err("must fail");
        assert!(err.contains("step must be a number"));
    }
}
