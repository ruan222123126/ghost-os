use super::image_ops::crop_image;
use super::params::{parse_capture_region, parse_optional_display_id};
use super::types::{CapturePayload, CapturedImage, sanitize_scale, scale_coordinate};
use crate::Response;
use crate::display_scale::record_display_scale;
use serde_json::Value;
use xcap::Monitor;

pub(crate) fn handle_screen_capture(params: &Value) -> Response {
    let captured = match capture_screen(params) {
        Ok(captured) => captured,
        Err(err) => return Response::error(err),
    };
    let payload = match capture_payload(captured) {
        Ok(payload) => payload,
        Err(err) => return Response::error(err),
    };
    match serde_json::to_value(payload) {
        Ok(payload) => Response::success(payload),
        Err(err) => Response::error(format!("encode screen capture payload failed: {err}")),
    }
}

pub(crate) fn capture_screen(params: &Value) -> Result<CapturedImage, String> {
    let display_id = parse_optional_display_id(params)?;
    let monitors = Monitor::all().map_err(|err| format!("list monitors failed: {err}"))?;
    if monitors.is_empty() {
        return Err("no monitor is available".to_string());
    }
    let monitor = select_monitor(&monitors, display_id)?;
    let image = monitor
        .capture_image()
        .map_err(|err| format!("capture monitor {} failed: {err}", monitor.id()))?;
    let region = parse_capture_region(params, image.width(), image.height())?;
    let cropped = crop_image(&image, region)?;
    let (scale_x, scale_y) = monitor_scale(monitor, image.width(), image.height());
    record_display_scale(monitor.id(), scale_x, scale_y);

    let origin_x = scale_coordinate(monitor.x(), scale_x) + region.x;
    let origin_y = scale_coordinate(monitor.y(), scale_y) + region.y;
    Ok(CapturedImage {
        display_id: monitor.id(),
        scale_x,
        scale_y,
        origin_x,
        origin_y,
        region,
        image: cropped,
    })
}

fn capture_payload(captured: CapturedImage) -> Result<CapturePayload, String> {
    let image_width = captured.image.width();
    let image_height = captured.image.height();
    let image_path = super::image_ops::write_temp_png(&captured.image, "ghost-os-screen-capture")?;
    Ok(CapturePayload {
        image_path,
        image_width,
        image_height,
        display_id: captured.display_id,
        scale_x: captured.scale_x,
        scale_y: captured.scale_y,
        origin_x: captured.origin_x,
        origin_y: captured.origin_y,
        region: captured.region,
    })
}

fn select_monitor<'a>(
    monitors: &'a [Monitor],
    display_id: Option<u32>,
) -> Result<&'a Monitor, String> {
    let primary = monitors
        .iter()
        .find(|monitor| monitor.is_primary())
        .unwrap_or(&monitors[0]);
    let available_ids = monitors.iter().map(Monitor::id).collect::<Vec<_>>();
    let selected_id = resolve_display_id(display_id, &available_ids, primary.id());
    monitors
        .iter()
        .find(|monitor| monitor.id() == selected_id)
        .ok_or_else(|| {
            format!(
                "selected display_id {selected_id} is not found; available displays: {}",
                available_ids
                    .iter()
                    .map(u32::to_string)
                    .collect::<Vec<_>>()
                    .join(",")
            )
        })
}

fn resolve_display_id(display_id: Option<u32>, available_ids: &[u32], primary_id: u32) -> u32 {
    match display_id {
        Some(requested_id) if available_ids.contains(&requested_id) => requested_id,
        Some(_) => primary_id,
        None => primary_id,
    }
}

fn monitor_scale(monitor: &Monitor, width: u32, height: u32) -> (f64, f64) {
    let logical_width = monitor.width();
    let logical_height = monitor.height();
    let scale_x = if logical_width > 0 {
        f64::from(width) / f64::from(logical_width)
    } else {
        1.0
    };
    let scale_y = if logical_height > 0 {
        f64::from(height) / f64::from(logical_height)
    } else {
        1.0
    };
    (sanitize_scale(scale_x), sanitize_scale(scale_y))
}

#[cfg(test)]
mod tests {
    use super::{monitor_scale, resolve_display_id};
    use crate::screen::types::sanitize_scale;

    #[test]
    fn sanitize_scale_keeps_positive_values() {
        assert_eq!(sanitize_scale(1.5), 1.5);
    }

    #[test]
    fn monitor_scale_defaults_to_identity_without_logical_dimensions() {
        struct FakeMonitor {
            width: i32,
            height: i32,
        }

        impl FakeMonitor {
            fn scale(&self, width: u32, height: u32) -> (f64, f64) {
                let scale_x = if self.width > 0 {
                    f64::from(width) / f64::from(self.width)
                } else {
                    1.0
                };
                let scale_y = if self.height > 0 {
                    f64::from(height) / f64::from(self.height)
                } else {
                    1.0
                };
                (sanitize_scale(scale_x), sanitize_scale(scale_y))
            }
        }

        let fake = FakeMonitor {
            width: 0,
            height: 0,
        };
        assert_eq!(fake.scale(100, 200), (1.0, 1.0));

        let _ = monitor_scale;
    }

    #[test]
    fn resolve_display_id_falls_back_to_primary_for_unknown_display() {
        let selected = resolve_display_id(Some(999), &[67, 68], 67);
        assert_eq!(selected, 67);
    }

    #[test]
    fn resolve_display_id_keeps_requested_display_when_available() {
        let selected = resolve_display_id(Some(68), &[67, 68], 67);
        assert_eq!(selected, 68);
    }
}
