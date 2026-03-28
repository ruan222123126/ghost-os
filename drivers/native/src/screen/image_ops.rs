use super::params::parse_required_region;
use super::types::{ImageCropPayload, ScreenRegion};
use crate::Response;
use crate::json_params::{optional_i32, optional_string, required_string};
use base64::Engine;
use serde_json::Value;
use std::fs::File;
use std::io::BufWriter;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};
use xcap::image::codecs::png::PngEncoder;
use xcap::image::{ColorType, ImageEncoder, ImageReader, RgbaImage, imageops};

pub(crate) fn handle_image_crop(params: &Value) -> Response {
    let image = match parse_image_from_params(params) {
        Ok(image) => image,
        Err(err) => return Response::error(err),
    };
    let region = match parse_required_region(params, "region", image.width(), image.height()) {
        Ok(region) => region,
        Err(err) => return Response::error(err),
    };
    let origin_x = match optional_i32(params, "origin_x") {
        Ok(value) => value.unwrap_or(0),
        Err(err) => return Response::error(err),
    };
    let origin_y = match optional_i32(params, "origin_y") {
        Ok(value) => value.unwrap_or(0),
        Err(err) => return Response::error(err),
    };
    let cropped = match crop_image(&image, region) {
        Ok(image) => image,
        Err(err) => return Response::error(err),
    };
    let image_base64 = match encode_png_base64(&cropped) {
        Ok(image_base64) => image_base64,
        Err(err) => return Response::error(err),
    };
    let payload = ImageCropPayload {
        image_base64,
        image_width: cropped.width(),
        image_height: cropped.height(),
        origin_x: origin_x + region.x,
        origin_y: origin_y + region.y,
        region,
    };
    match serde_json::to_value(payload) {
        Ok(payload) => Response::success(payload),
        Err(err) => Response::error(format!("encode image crop payload failed: {err}")),
    }
}

pub(crate) fn crop_image(image: &RgbaImage, region: ScreenRegion) -> Result<RgbaImage, String> {
    let x = u32::try_from(region.x).map_err(|_| "region.x must be non-negative".to_string())?;
    let y = u32::try_from(region.y).map_err(|_| "region.y must be non-negative".to_string())?;
    let width =
        u32::try_from(region.width).map_err(|_| "region.width must be positive".to_string())?;
    let height =
        u32::try_from(region.height).map_err(|_| "region.height must be positive".to_string())?;

    Ok(imageops::crop_imm(image, x, y, width, height).to_image())
}

pub(crate) fn decode_png_base64(image_base64: &str) -> Result<RgbaImage, String> {
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(image_base64.trim())
        .map_err(|err| format!("decode image_base64 failed: {err}"))?;
    xcap::image::load_from_memory(&bytes)
        .map(|image| image.to_rgba8())
        .map_err(|err| format!("decode image failed: {err}"))
}

pub(crate) fn encode_png_base64(image: &RgbaImage) -> Result<String, String> {
    let bytes = encode_png_bytes(image)?;
    Ok(base64::engine::general_purpose::STANDARD.encode(bytes))
}

pub(crate) fn load_image_from_path(path: &str) -> Result<RgbaImage, String> {
    let image_path = Path::new(path);
    if !image_path.exists() {
        return Err(format!("image_path does not exist: {path}"));
    }
    ImageReader::open(image_path)
        .map_err(|err| format!("open image failed: {err}"))?
        .decode()
        .map(|image| image.to_rgba8())
        .map_err(|err| format!("decode image failed: {err}"))
}

pub(crate) fn parse_image_from_params(params: &Value) -> Result<RgbaImage, String> {
    if let Some(image_path) = optional_string(params, "image_path")? {
        return load_image_from_path(&image_path);
    }
    let image_base64 = required_string(params, "image_base64")?;
    decode_png_base64(&image_base64)
}

pub(crate) fn write_temp_png(image: &RgbaImage, prefix: &str) -> Result<String, String> {
    let path = unique_temp_path(prefix, "png");
    write_png_file(image, &path)?;
    Ok(path.to_string_lossy().into_owned())
}

fn encode_png_bytes(image: &RgbaImage) -> Result<Vec<u8>, String> {
    let mut buf = Vec::new();
    let encoder = PngEncoder::new(&mut buf);
    encoder
        .write_image(
            image.as_raw(),
            image.width(),
            image.height(),
            ColorType::Rgba8.into(),
        )
        .map_err(|err| format!("encode png failed: {err}"))?;
    Ok(buf)
}

fn write_png_file(image: &RgbaImage, path: &Path) -> Result<(), String> {
    let file = File::create(path).map_err(|err| format!("create temp image failed: {err}"))?;
    let mut writer = BufWriter::new(file);
    let encoder = PngEncoder::new(&mut writer);
    encoder
        .write_image(
            image.as_raw(),
            image.width(),
            image.height(),
            ColorType::Rgba8.into(),
        )
        .map_err(|err| format!("write temp image failed: {err}"))
}

fn unique_temp_path(prefix: &str, extension: &str) -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let pid = std::process::id();
    std::env::temp_dir().join(format!("{prefix}-{pid}-{nanos}.{extension}"))
}

#[cfg(test)]
mod tests {
    use super::{crop_image, decode_png_base64, encode_png_base64};
    use crate::screen::types::ScreenRegion;
    use xcap::image::{Rgba, RgbaImage};

    #[test]
    fn crop_image_returns_requested_region() {
        let mut image = RgbaImage::from_pixel(4, 3, Rgba([0, 0, 0, 255]));
        image.put_pixel(1, 1, Rgba([255, 0, 0, 255]));
        let cropped = crop_image(
            &image,
            ScreenRegion {
                x: 1,
                y: 1,
                width: 2,
                height: 1,
            },
        )
        .expect("crop must succeed");
        assert_eq!(cropped.width(), 2);
        assert_eq!(cropped.height(), 1);
        assert_eq!(cropped.get_pixel(0, 0)[0], 255);
    }

    #[test]
    fn png_base64_round_trips() {
        let image = RgbaImage::from_pixel(2, 2, Rgba([10, 20, 30, 255]));
        let encoded = encode_png_base64(&image).expect("must encode");
        let decoded = decode_png_base64(&encoded).expect("must decode");
        assert_eq!(decoded.width(), 2);
        assert_eq!(decoded.height(), 2);
    }
}
