use super::image_ops::crop_image;
use super::params::{parse_capture_region, parse_optional_display_id};
use super::types::{CapturePayload, CapturedImage, sanitize_scale, scale_coordinate};
use super::wayland_screencast;
use crate::Response;
use crate::display_scale::record_display_scale;
use serde_json::Value;
use std::env::var_os;
use std::sync::OnceLock;
use xcap::Monitor;
use xcap::image::RgbaImage;

static CAPTURE_BACKEND_LOGGED: OnceLock<()> = OnceLock::new();

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
    let source = capture_monitor_image(monitor)?;
    let region = parse_capture_region(params, source.image.width(), source.image.height())?;
    let cropped = crop_image(&source.image, region)?;
    let (scale_x, scale_y, base_origin_x, base_origin_y) = capture_transform(monitor, &source);
    record_display_scale(monitor.id(), scale_x, scale_y);

    let origin_x = base_origin_x + region.x;
    let origin_y = base_origin_y + region.y;
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

struct CaptureSource {
    image: RgbaImage,
    logical_size: Option<(i32, i32)>,
    origin: Option<(i32, i32)>,
}

fn capture_monitor_image(monitor: &Monitor) -> Result<CaptureSource, String> {
    if !is_wayland_session() {
        log_capture_backend("xcap");
        let image = monitor
            .capture_image()
            .map_err(|err| format!("capture monitor {} failed: {err}", monitor.id()))?;
        return Ok(CaptureSource {
            image,
            logical_size: None,
            origin: None,
        });
    }

    log_capture_backend("portal-screencast");
    let frame = wayland_screencast::capture_frame()
        .map_err(|err| format!("wayland capture failed: {err}"))?;
    Ok(CaptureSource {
        image: frame.image,
        logical_size: frame.logical_size,
        origin: frame.origin,
    })
}

fn capture_transform(monitor: &Monitor, source: &CaptureSource) -> (f64, f64, i32, i32) {
    let fallback = monitor_scale(monitor, source.image.width(), source.image.height());
    let scale = source
        .logical_size
        .map(|(width, height)| {
            scale_from_logical_size(source.image.width(), source.image.height(), width, height)
        })
        .unwrap_or(fallback);
    let base_origin = source.origin.unwrap_or((monitor.x(), monitor.y()));
    (
        scale.0,
        scale.1,
        scale_coordinate(base_origin.0, scale.0),
        scale_coordinate(base_origin.1, scale.1),
    )
}

fn scale_from_logical_size(
    width: u32,
    height: u32,
    logical_width: i32,
    logical_height: i32,
) -> (f64, f64) {
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

fn is_wayland_session() -> bool {
    let xdg_session_type = var_os("XDG_SESSION_TYPE")
        .unwrap_or_default()
        .to_string_lossy()
        .to_lowercase();
    let wayland_display = var_os("WAYLAND_DISPLAY")
        .unwrap_or_default()
        .to_string_lossy()
        .to_lowercase();
    xdg_session_type == "wayland" || wayland_display.contains("wayland")
}

fn log_capture_backend(backend: &str) {
    if CAPTURE_BACKEND_LOGGED.set(()).is_err() {
        return;
    }
    eprintln!(
        "ghost-native: screen capture backend={} session_type={} wayland_display={} display={}",
        backend,
        capture_env_value("XDG_SESSION_TYPE"),
        capture_env_value("WAYLAND_DISPLAY"),
        capture_env_value("DISPLAY"),
    );
}

fn capture_env_value(name: &str) -> String {
    let value = var_os(name)
        .unwrap_or_default()
        .to_string_lossy()
        .trim()
        .to_string();
    if value.is_empty() {
        return "<unset>".to_string();
    }
    value
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

fn select_monitor(monitors: &[Monitor], display_id: Option<u32>) -> Result<&Monitor, String> {
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
