use super::image_ops::load_image_from_path;
use super::match_engine::find_template_matches_with_scale;
use super::params::parse_scale_range;
use crate::Response;
use crate::json_params::{optional_f64, optional_i32, optional_usize, required_string};
use serde_json::{Value, json};
use std::path::Path;
use xcap::image::{ImageReader, RgbaImage};

pub(crate) fn handle_template_match_image(params: &Value) -> Response {
    let image_path = match required_string(params, "image_path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let image = match load_image_from_path(&image_path) {
        Ok(image) => image,
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
    let template_path = match required_string(params, "template_path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let threshold = match optional_f64(params, "threshold") {
        Ok(Some(value)) => value.clamp(0.0, 1.0),
        Ok(None) => 0.88,
        Err(err) => return Response::error(err),
    };
    let max_results = match optional_usize(params, "max_results") {
        Ok(Some(value)) if value > 0 => value,
        Ok(Some(_)) => return Response::error("max_results must be greater than 0".to_string()),
        Ok(None) => 5,
        Err(err) => return Response::error(err),
    };
    let scale_range = match parse_scale_range(params) {
        Ok(range) => range,
        Err(err) => return Response::error(err),
    };
    let template = match load_template_image(&template_path) {
        Ok(image) => image,
        Err(err) => return Response::error(err),
    };
    let matches = match find_template_matches_with_scale(
        &image,
        &template,
        origin_x,
        origin_y,
        threshold,
        max_results,
        scale_range,
    ) {
        Ok(matches) => matches,
        Err(err) => return Response::error(err),
    };
    Response::success(json!({ "matches": matches }))
}

fn load_template_image(path: &str) -> Result<RgbaImage, String> {
    let template_path = Path::new(path);
    if !template_path.exists() {
        return Err(format!("template_path does not exist: {path}"));
    }

    ImageReader::open(template_path)
        .map_err(|err| format!("open template image failed: {err}"))?
        .decode()
        .map(|image| image.to_rgba8())
        .map_err(|err| format!("decode template image failed: {err}"))
}
