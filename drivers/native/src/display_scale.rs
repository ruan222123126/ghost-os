use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};

static DISPLAY_SCALES: OnceLock<Mutex<HashMap<u32, (f64, f64)>>> = OnceLock::new();

pub(crate) fn record_display_scale(display_id: u32, scale_x: f64, scale_y: f64) {
    if !(scale_x.is_finite() && scale_y.is_finite()) {
        return;
    }
    let scale_x = if scale_x <= 0.0 { 1.0 } else { scale_x };
    let scale_y = if scale_y <= 0.0 { 1.0 } else { scale_y };
    let map = DISPLAY_SCALES.get_or_init(|| Mutex::new(HashMap::new()));
    if let Ok(mut guard) = map.lock() {
        guard.insert(display_id, (scale_x, scale_y));
    }
}

pub(crate) fn lookup_display_scale(display_id: u32) -> Option<(f64, f64)> {
    let map = DISPLAY_SCALES.get_or_init(|| Mutex::new(HashMap::new()));
    let guard = map.lock().ok()?;
    guard.get(&display_id).copied()
}
