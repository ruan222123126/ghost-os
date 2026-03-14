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
    let image_base64 = super::image_ops::encode_png_base64(&captured.image)?;
    Ok(CapturePayload {
        image_base64,
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
    if let Some(id) = display_id {
        return monitors
            .iter()
            .find(|monitor| monitor.id() == id)
            .ok_or_else(|| {
                format!(
                    "display_id {id} is not found; available displays: {}",
                    monitors
                        .iter()
                        .map(|monitor| monitor.id().to_string())
                        .collect::<Vec<_>>()
                        .join(",")
                )
            });
    }
    Ok(monitors
        .iter()
        .find(|monitor| monitor.is_primary())
        .unwrap_or(&monitors[0]))
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
    use super::monitor_scale;
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
}
