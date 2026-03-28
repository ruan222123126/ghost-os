use serde::{Deserialize, Serialize};
use xcap::image::RgbaImage;

#[derive(Clone, Copy, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub(crate) struct ScreenRegion {
    pub(crate) x: i32,
    pub(crate) y: i32,
    pub(crate) width: i32,
    pub(crate) height: i32,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub(crate) struct ScreenPoint {
    pub(crate) x: i32,
    pub(crate) y: i32,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub(crate) struct ScreenBoundingBox {
    pub(crate) x: i32,
    pub(crate) y: i32,
    pub(crate) width: i32,
    pub(crate) height: i32,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub(crate) struct OcrItem {
    pub(crate) text: String,
    pub(crate) confidence: f64,
    pub(crate) bbox: ScreenBoundingBox,
    pub(crate) center: ScreenPoint,
}

#[derive(Clone, Debug, Serialize, Deserialize, PartialEq)]
pub(crate) struct IconMatch {
    pub(crate) score: f64,
    pub(crate) bbox: ScreenBoundingBox,
    pub(crate) center: ScreenPoint,
    pub(crate) scale: f64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct CapturePayload {
    pub(crate) image_path: String,
    pub(crate) image_width: u32,
    pub(crate) image_height: u32,
    pub(crate) display_id: u32,
    pub(crate) scale_x: f64,
    pub(crate) scale_y: f64,
    pub(crate) origin_x: i32,
    pub(crate) origin_y: i32,
    pub(crate) region: ScreenRegion,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub(crate) struct ImageCropPayload {
    pub(crate) image_base64: String,
    pub(crate) image_width: u32,
    pub(crate) image_height: u32,
    pub(crate) origin_x: i32,
    pub(crate) origin_y: i32,
    pub(crate) region: ScreenRegion,
}

pub(crate) struct CapturedImage {
    pub(crate) display_id: u32,
    pub(crate) scale_x: f64,
    pub(crate) scale_y: f64,
    pub(crate) origin_x: i32,
    pub(crate) origin_y: i32,
    pub(crate) region: ScreenRegion,
    pub(crate) image: RgbaImage,
}

pub(crate) fn sanitize_scale(value: f64) -> f64 {
    if value.is_finite() && value > 0.0 {
        value
    } else {
        1.0
    }
}

pub(crate) fn scale_coordinate(value: i32, scale: f64) -> i32 {
    if scale <= 0.0 || !scale.is_finite() {
        return value;
    }
    ((value as f64) * scale).round() as i32
}

pub(crate) fn unscale_coordinate(value: i32, scale: f64) -> i32 {
    if scale.is_finite() && scale > 0.0 && (scale - 1.0).abs() > 1e-6 {
        ((value as f64) / scale).round() as i32
    } else {
        value
    }
}

#[cfg(test)]
mod tests {
    use super::{sanitize_scale, scale_coordinate, unscale_coordinate};

    #[test]
    fn sanitize_scale_defaults_invalid_values() {
        assert_eq!(sanitize_scale(0.0), 1.0);
        assert_eq!(sanitize_scale(-1.0), 1.0);
    }

    #[test]
    fn coordinate_scaling_round_trips() {
        let scaled = scale_coordinate(30, 1.5);
        assert_eq!(scaled, 45);
        assert_eq!(unscale_coordinate(scaled, 1.5), 30);
    }
}
