use crate::Response;
use crate::display_scale::lookup_display_scale;
use crate::json_params::{optional_display_id, optional_i32};
use crate::screen::types::{sanitize_scale, unscale_coordinate};
use serde_json::{Value, json};

#[cfg(target_os = "linux")]
use std::process::Command;

pub(crate) fn handle_mouse_scroll(params: &Value) -> Response {
    let request = match parse_scroll_request(params) {
        Ok(request) => request,
        Err(err) => return Response::error(err),
    };
    let point = request
        .point
        .map(|(x, y)| resolve_scroll_point(x, y, request.display_id));
    if let Err(err) = perform_mouse_scroll(point, request.delta_x, request.delta_y) {
        return Response::error(err);
    }

    let (resolved_x, resolved_y, scale_x, scale_y) = match point {
        Some((x, y, sx, sy)) => (Some(x), Some(y), sx, sy),
        None => (None, None, 1.0, 1.0),
    };
    Response::success(json!({
        "scrolled": true,
        "x": resolved_x,
        "y": resolved_y,
        "delta_x": request.delta_x,
        "delta_y": request.delta_y,
        "scale_x": scale_x,
        "scale_y": scale_y,
    }))
}

#[derive(Debug)]
struct ScrollRequest {
    point: Option<(i32, i32)>,
    delta_x: i32,
    delta_y: i32,
    display_id: Option<u32>,
}

fn parse_scroll_request(params: &Value) -> Result<ScrollRequest, String> {
    let delta_x = optional_i32(params, "delta_x")?.unwrap_or(0);
    let delta_y = optional_i32(params, "delta_y")?.unwrap_or(0);
    if delta_x == 0 && delta_y == 0 {
        return Err("delta_x or delta_y is required".to_string());
    }
    let x = optional_i32(params, "x")?;
    let y = optional_i32(params, "y")?;
    let point = match (x, y) {
        (Some(x), Some(y)) => Some((x, y)),
        (None, None) => None,
        _ => return Err("x and y must be provided together".to_string()),
    };
    Ok(ScrollRequest {
        point,
        delta_x,
        delta_y,
        display_id: optional_display_id(params)?,
    })
}

fn resolve_scroll_point(x: i32, y: i32, display_id: Option<u32>) -> (i32, i32, f64, f64) {
    let (scale_x, scale_y) = display_id
        .and_then(lookup_display_scale)
        .map(|(sx, sy)| (sanitize_scale(sx), sanitize_scale(sy)))
        .unwrap_or((1.0, 1.0));
    (
        unscale_coordinate(x, scale_x),
        unscale_coordinate(y, scale_y),
        scale_x,
        scale_y,
    )
}

#[cfg(target_os = "linux")]
fn perform_mouse_scroll(
    point: Option<(i32, i32, f64, f64)>,
    delta_x: i32,
    delta_y: i32,
) -> Result<(), String> {
    if super::window_guard::is_wayland_session() {
        return Err("mouse scroll is not supported on Wayland".to_string());
    }
    if let Some((x, y, _, _)) = point {
        let status = Command::new("xdotool")
            .args(["mousemove", "--sync", &x.to_string(), &y.to_string()])
            .status()
            .map_err(|err| format!("spawn xdotool failed: {err}"))?;
        if !status.success() {
            return Err(format!(
                "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
            ));
        }
    }
    run_scroll_clicks(delta_y, "5", "4")?;
    run_scroll_clicks(delta_x, "7", "6")?;
    Ok(())
}

#[cfg(target_os = "linux")]
fn run_scroll_clicks(
    delta: i32,
    positive_button: &str,
    negative_button: &str,
) -> Result<(), String> {
    let amount = delta.unsigned_abs();
    let button = if delta > 0 {
        positive_button
    } else {
        negative_button
    };
    for _ in 0..amount {
        let status = Command::new("xdotool")
            .args(["click", button])
            .status()
            .map_err(|err| format!("spawn xdotool failed: {err}"))?;
        if !status.success() {
            return Err(format!(
                "xdotool exited with status {status}; ensure xdotool is installed and graphical session is active"
            ));
        }
    }
    Ok(())
}

#[cfg(not(target_os = "linux"))]
fn perform_mouse_scroll(
    _point: Option<(i32, i32, f64, f64)>,
    _delta_x: i32,
    _delta_y: i32,
) -> Result<(), String> {
    Err("mouse scroll is not supported on this platform".to_string())
}

#[cfg(test)]
mod tests {
    use super::{parse_scroll_request, resolve_scroll_point};
    use crate::display_scale::record_display_scale;
    use serde_json::json;

    #[test]
    fn parse_scroll_request_requires_delta() {
        let err = parse_scroll_request(&json!({})).expect_err("must fail");
        assert!(err.contains("delta_x or delta_y"));
    }

    #[test]
    fn resolve_scroll_point_uses_display_scale() {
        record_display_scale(11, 2.0, 1.5);
        let (x, y, sx, sy) = resolve_scroll_point(200, 150, Some(11));
        assert_eq!((x, y), (100, 100));
        assert_eq!((sx, sy), (2.0, 1.5));
    }
}
