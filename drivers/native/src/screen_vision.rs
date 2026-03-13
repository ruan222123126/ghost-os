use crate::Response;
use crate::display_scale::record_display_scale;
use serde::Serialize;
use serde_json::{Value, json};
use std::cmp::Ordering;
use std::fs;
use std::path::Path;
use std::process::Command;
use std::time::{SystemTime, UNIX_EPOCH};
use xcap::Monitor;
use xcap::image::{DynamicImage, ImageReader, Pixel, RgbaImage};
use xcap::image::imageops::{FilterType, resize};

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    match action {
        "SCREEN_OCR" => Some(handle_screen_ocr(params)),
        "ICON_MATCH" => Some(handle_icon_match(params)),
        _ => None,
    }
}

fn handle_screen_ocr(params: &Value) -> Response {
    let capture = match capture_screen_region(params) {
        Ok(capture) => capture,
        Err(err) => return Response::error(err),
    };

    let languages = match parse_languages(params) {
        Ok(languages) => languages,
        Err(err) => return Response::error(err),
    };
    let min_confidence = match parse_optional_f64(params, "min_confidence") {
        Ok(Some(value)) => value.clamp(0.0, 1.0),
        Ok(None) => 0.75,
        Err(err) => return Response::error(err),
    };

    let output = match run_tesseract_ocr(&capture.image, &languages) {
        Ok(output) => output,
        Err(err) => return Response::error(err),
    };
    let mut items = match parse_tesseract_tsv(
        &output,
        capture.monitor_x + capture.region.x,
        capture.monitor_y + capture.region.y,
        min_confidence,
    ) {
        Ok(items) => items,
        Err(err) => return Response::error(err),
    };
    items.sort_by(item_position_order);

    Response::success(json!({
        "display_id": capture.display_id,
        "image_width": capture.full_width,
        "image_height": capture.full_height,
        "scale_x": capture.scale_x,
        "scale_y": capture.scale_y,
        "region": capture.region,
        "items": items,
    }))
}

fn handle_icon_match(params: &Value) -> Response {
    let capture = match capture_screen_region(params) {
        Ok(capture) => capture,
        Err(err) => return Response::error(err),
    };
    let template_path = match parse_required_string(params, "template_path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let threshold = match parse_optional_f64(params, "threshold") {
        Ok(Some(value)) => value.clamp(0.0, 1.0),
        Ok(None) => 0.88,
        Err(err) => return Response::error(err),
    };
    let max_results = match parse_optional_usize(params, "max_results") {
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
        &capture.image,
        &template,
        capture.monitor_x + capture.region.x,
        capture.monitor_y + capture.region.y,
        threshold,
        max_results,
        scale_range,
    ) {
        Ok(matches) => matches,
        Err(err) => return Response::error(err),
    };

    Response::success(json!({
        "display_id": capture.display_id,
        "image_width": capture.full_width,
        "image_height": capture.full_height,
        "scale_x": capture.scale_x,
        "scale_y": capture.scale_y,
        "region": capture.region,
        "matches": matches,
    }))
}

#[derive(Clone, Copy, Debug, Serialize, PartialEq, Eq)]
struct ScreenRegion {
    x: i32,
    y: i32,
    width: i32,
    height: i32,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
struct ScreenPoint {
    x: i32,
    y: i32,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
struct ScreenBoundingBox {
    x: i32,
    y: i32,
    width: i32,
    height: i32,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
struct OcrItem {
    text: String,
    confidence: f64,
    bbox: ScreenBoundingBox,
    center: ScreenPoint,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
struct IconMatch {
    score: f64,
    bbox: ScreenBoundingBox,
    center: ScreenPoint,
    scale: f64,
}

struct ScreenCapture {
    display_id: u32,
    monitor_x: i32,
    monitor_y: i32,
    full_width: u32,
    full_height: u32,
    scale_x: f64,
    scale_y: f64,
    region: ScreenRegion,
    image: RgbaImage,
}

fn capture_screen_region(params: &Value) -> Result<ScreenCapture, String> {
    let display_id = parse_optional_display_id(params)?;

    let monitors = Monitor::all().map_err(|err| format!("list monitors failed: {err}"))?;
    if monitors.is_empty() {
        return Err("no monitor is available".to_string());
    }

    let monitor = if let Some(id) = display_id {
        monitors
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
            })?
    } else {
        monitors
            .iter()
            .find(|monitor| monitor.is_primary())
            .unwrap_or(&monitors[0])
    };

    let image = monitor
        .capture_image()
        .map_err(|err| format!("capture monitor {} failed: {err}", monitor.id()))?;
    let full_width = image.width();
    let full_height = image.height();
    let (scale_x, scale_y) = monitor_scale(&monitor, full_width, full_height);
    record_display_scale(monitor.id(), scale_x, scale_y);
    let region = parse_region(params, full_width, full_height)?;
    let cropped = crop_region(&image, region)?;

    Ok(ScreenCapture {
        display_id: monitor.id(),
        monitor_x: scale_coordinate(monitor.x(), scale_x),
        monitor_y: scale_coordinate(monitor.y(), scale_y),
        full_width,
        full_height,
        scale_x,
        scale_y,
        region,
        image: cropped,
    })
}

fn crop_region(image: &RgbaImage, region: ScreenRegion) -> Result<RgbaImage, String> {
    let x = u32::try_from(region.x).map_err(|_| "region.x must be non-negative".to_string())?;
    let y = u32::try_from(region.y).map_err(|_| "region.y must be non-negative".to_string())?;
    let width =
        u32::try_from(region.width).map_err(|_| "region.width must be positive".to_string())?;
    let height =
        u32::try_from(region.height).map_err(|_| "region.height must be positive".to_string())?;

    Ok(DynamicImage::ImageRgba8(image.clone())
        .crop_imm(x, y, width, height)
        .to_rgba8())
}

fn monitor_scale(monitor: &Monitor, full_width: u32, full_height: u32) -> (f64, f64) {
    let logical_width = monitor.width();
    let logical_height = monitor.height();
    let scale_x = if logical_width > 0 {
        f64::from(full_width) / f64::from(logical_width)
    } else {
        1.0
    };
    let scale_y = if logical_height > 0 {
        f64::from(full_height) / f64::from(logical_height)
    } else {
        1.0
    };
    (sanitize_scale(scale_x), sanitize_scale(scale_y))
}

fn sanitize_scale(value: f64) -> f64 {
    if value.is_finite() && value > 0.0 {
        value
    } else {
        1.0
    }
}

fn scale_coordinate(value: i32, scale: f64) -> i32 {
    if scale <= 0.0 || !scale.is_finite() {
        return value;
    }
    ((value as f64) * scale).round() as i32
}

fn parse_region(params: &Value, full_width: u32, full_height: u32) -> Result<ScreenRegion, String> {
    let default_region = ScreenRegion {
        x: 0,
        y: 0,
        width: i32::try_from(full_width).map_err(|_| "screen width is too large".to_string())?,
        height: i32::try_from(full_height).map_err(|_| "screen height is too large".to_string())?,
    };

    let Some(raw) = params.get("region") else {
        return Ok(default_region);
    };
    let region = raw
        .as_object()
        .ok_or_else(|| "region must be an object".to_string())?;

    let x = parse_region_field(region, "x")?;
    let y = parse_region_field(region, "y")?;
    let width = parse_region_field(region, "width")?;
    let height = parse_region_field(region, "height")?;

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
    if x + width > full_width_i32 || y + height > full_height_i32 {
        return Err(format!(
            "region {}x{} at ({x},{y}) exceeds screen bounds {}x{}",
            width, height, full_width, full_height
        ));
    }

    Ok(ScreenRegion {
        x,
        y,
        width,
        height,
    })
}

fn parse_region_field(region: &serde_json::Map<String, Value>, field: &str) -> Result<i32, String> {
    let raw = region
        .get(field)
        .ok_or_else(|| format!("region.{field} is required"))?;
    let value = raw
        .as_i64()
        .ok_or_else(|| format!("region.{field} must be an integer"))?;
    i32::try_from(value).map_err(|_| format!("region.{field} is out of i32 range"))
}

fn parse_optional_display_id(params: &Value) -> Result<Option<u32>, String> {
    let Some(raw) = params.get("display_id") else {
        return Ok(None);
    };

    let value = raw
        .as_u64()
        .ok_or_else(|| "display_id must be a non-negative integer".to_string())?;
    let display_id = u32::try_from(value).map_err(|_| "display_id is too large".to_string())?;
    Ok(Some(display_id))
}

fn parse_required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn parse_optional_f64(params: &Value, field: &str) -> Result<Option<f64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_f64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a number"))
}

fn parse_optional_usize(params: &Value, field: &str) -> Result<Option<usize>, String> {
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

#[derive(Clone, Copy, Debug)]
struct ScaleRange {
    min: f64,
    max: f64,
    step: f64,
}

fn parse_scale_range(params: &Value) -> Result<Option<ScaleRange>, String> {
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

fn parse_languages(params: &Value) -> Result<Vec<String>, String> {
    let mut languages = Vec::new();
    if let Some(raw) = params.get("languages") {
        let items = raw
            .as_array()
            .ok_or_else(|| "languages must be an array".to_string())?;
        for item in items {
            let language = item
                .as_str()
                .map(str::trim)
                .filter(|value| !value.is_empty())
                .ok_or_else(|| "languages must contain only non-empty strings".to_string())?;
            languages.push(map_tesseract_language(language));
        }
    }
    if languages.is_empty() {
        languages.push(map_tesseract_language("zh"));
        languages.push(map_tesseract_language("en"));
    }
    languages.sort();
    languages.dedup();
    Ok(languages)
}

fn map_tesseract_language(language: &str) -> String {
    match language.trim().to_ascii_lowercase().as_str() {
        "zh" | "zh-cn" | "zh-hans" | "ch" | "chi_sim" => "chi_sim".to_string(),
        "en" | "eng" => "eng".to_string(),
        other => other.to_string(),
    }
}

fn run_tesseract_ocr(image: &RgbaImage, languages: &[String]) -> Result<String, String> {
    let input_path = unique_temp_path("ghost-os-screen-ocr", "png");
    write_image_png(image, &input_path)?;

    let language_arg = if languages.is_empty() {
        "eng".to_string()
    } else {
        languages.join("+")
    };
    let output = Command::new("tesseract")
        .args([
            input_path.to_string_lossy().as_ref(),
            "stdout",
            "-l",
            &language_arg,
            "--psm",
            "11",
            "tsv",
        ])
        .output();

    let _ = fs::remove_file(&input_path);

    match output {
        Ok(result) if result.status.success() => {
            Ok(String::from_utf8_lossy(&result.stdout).into_owned())
        }
        Ok(result) => {
            let stderr = String::from_utf8_lossy(&result.stderr).trim().to_string();
            if stderr.is_empty() {
                Err(format!("tesseract exited with status {}", result.status))
            } else {
                Err(format!("tesseract failed: {stderr}"))
            }
        }
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
            Err("tesseract is not installed".to_string())
        }
        Err(err) => Err(format!("spawn tesseract failed: {err}")),
    }
}

fn write_image_png(image: &RgbaImage, path: &Path) -> Result<(), String> {
    DynamicImage::ImageRgba8(image.clone())
        .save(path)
        .map_err(|err| format!("write temp image failed: {err}"))
}

fn unique_temp_path(prefix: &str, extension: &str) -> std::path::PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or(0);
    let pid = std::process::id();
    std::env::temp_dir().join(format!("{prefix}-{pid}-{nanos}.{extension}"))
}

fn parse_tesseract_tsv(
    input: &str,
    offset_x: i32,
    offset_y: i32,
    min_confidence: f64,
) -> Result<Vec<OcrItem>, String> {
    let mut items = Vec::new();
    for (index, line) in input.lines().enumerate() {
        if index == 0 || line.trim().is_empty() {
            continue;
        }
        let cols: Vec<&str> = line.split('\t').collect();
        if cols.len() < 12 {
            continue;
        }

        let text = cols[11].trim();
        if text.is_empty() {
            continue;
        }

        let confidence = match cols[10].trim().parse::<f64>() {
            Ok(value) if value >= 0.0 => (value / 100.0).clamp(0.0, 1.0),
            _ => continue,
        };
        if confidence < min_confidence {
            continue;
        }

        let left = cols[6]
            .trim()
            .parse::<i32>()
            .map_err(|err| format!("decode tesseract left failed: {err}"))?;
        let top = cols[7]
            .trim()
            .parse::<i32>()
            .map_err(|err| format!("decode tesseract top failed: {err}"))?;
        let width = cols[8]
            .trim()
            .parse::<i32>()
            .map_err(|err| format!("decode tesseract width failed: {err}"))?;
        let height = cols[9]
            .trim()
            .parse::<i32>()
            .map_err(|err| format!("decode tesseract height failed: {err}"))?;
        if width <= 0 || height <= 0 {
            continue;
        }

        let x = offset_x + left;
        let y = offset_y + top;
        items.push(OcrItem {
            text: text.to_string(),
            confidence,
            bbox: ScreenBoundingBox {
                x,
                y,
                width,
                height,
            },
            center: ScreenPoint {
                x: x + (width / 2),
                y: y + (height / 2),
            },
        });
    }
    Ok(items)
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

fn find_template_matches(
    image: &RgbaImage,
    template: &RgbaImage,
    offset_x: i32,
    offset_y: i32,
    threshold: f64,
    max_results: usize,
) -> Result<Vec<IconMatch>, String> {
    if template.width() == 0 || template.height() == 0 {
        return Err("template image must not be empty".to_string());
    }
    if template.width() > image.width() || template.height() > image.height() {
        return Err("template image is larger than search region".to_string());
    }

    let haystack = to_grayscale(image);
    let needle = to_grayscale(template);
    let haystack_width =
        usize::try_from(image.width()).map_err(|_| "image width is too large".to_string())?;
    let haystack_height =
        usize::try_from(image.height()).map_err(|_| "image height is too large".to_string())?;
    let template_width =
        usize::try_from(template.width()).map_err(|_| "template width is too large".to_string())?;
    let template_height = usize::try_from(template.height())
        .map_err(|_| "template height is too large".to_string())?;

    let mut candidates = Vec::new();
    for y in 0..=(haystack_height - template_height) {
        for x in 0..=(haystack_width - template_width) {
            let score = template_similarity(
                &haystack,
                haystack_width,
                &needle,
                template_width,
                template_height,
                x,
                y,
            );
            if score < threshold {
                continue;
            }
            let x_i32 = i32::try_from(x).map_err(|_| "match x is too large".to_string())?;
            let y_i32 = i32::try_from(y).map_err(|_| "match y is too large".to_string())?;
            let width_i32 = i32::try_from(template_width)
                .map_err(|_| "template width is too large".to_string())?;
            let height_i32 = i32::try_from(template_height)
                .map_err(|_| "template height is too large".to_string())?;
            let abs_x = offset_x + x_i32;
            let abs_y = offset_y + y_i32;
            candidates.push(IconMatch {
                score,
                bbox: ScreenBoundingBox {
                    x: abs_x,
                    y: abs_y,
                    width: width_i32,
                    height: height_i32,
                },
                center: ScreenPoint {
                    x: abs_x + (width_i32 / 2),
                    y: abs_y + (height_i32 / 2),
                },
                scale: 1.0,
            });
        }
    }

    candidates.sort_by(
        |a, b| match b.score.partial_cmp(&a.score).unwrap_or(Ordering::Equal) {
            Ordering::Equal => item_position_order_bbox(&a.bbox, &b.bbox),
            other => other,
        },
    );

    let mut filtered = Vec::new();
    for candidate in candidates {
        if filtered
            .iter()
            .any(|kept: &IconMatch| bbox_overlap_ratio(&kept.bbox, &candidate.bbox) > 0.5)
        {
            continue;
        }
        filtered.push(candidate);
        if filtered.len() >= max_results {
            break;
        }
    }
    Ok(filtered)
}

fn find_template_matches_with_scale(
    image: &RgbaImage,
    template: &RgbaImage,
    offset_x: i32,
    offset_y: i32,
    threshold: f64,
    max_results: usize,
    scale_range: Option<ScaleRange>,
) -> Result<Vec<IconMatch>, String> {
    let Some(range) = scale_range else {
        return find_template_matches(
            image,
            template,
            offset_x,
            offset_y,
            threshold,
            max_results,
        );
    };

    let scales = build_scale_steps(range)?;
    let per_scale_max = max_results.saturating_mul(2).clamp(1, 20);
    let mut candidates = Vec::new();

    for scale in scales {
        let scaled = scale_template(template, scale)?;
        if scaled.width() == 0 || scaled.height() == 0 {
            continue;
        }
        if scaled.width() > image.width() || scaled.height() > image.height() {
            continue;
        }
        let mut matches = find_template_matches(
            image,
            &scaled,
            offset_x,
            offset_y,
            threshold,
            per_scale_max,
        )?;
        for item in &mut matches {
            item.scale = scale;
        }
        candidates.extend(matches);
    }

    Ok(merge_icon_matches(candidates, max_results))
}

fn build_scale_steps(range: ScaleRange) -> Result<Vec<f64>, String> {
    let mut scales = Vec::new();
    let mut value = range.min;
    let mut steps = 0usize;
    let max_steps = 15usize;
    while value <= range.max + 1e-6 {
        scales.push(value);
        value += range.step;
        steps += 1;
        if steps > max_steps {
            return Err("scale_range yields too many steps; increase step size".to_string());
        }
    }
    if scales.is_empty() {
        scales.push(1.0);
    }
    Ok(scales)
}

fn scale_template(template: &RgbaImage, scale: f64) -> Result<RgbaImage, String> {
    if !scale.is_finite() || scale <= 0.0 {
        return Err("scale value must be a positive number".to_string());
    }
    if (scale - 1.0).abs() < 1e-6 {
        return Ok(template.clone());
    }
    let width = (template.width() as f64 * scale).round() as u32;
    let height = (template.height() as f64 * scale).round() as u32;
    if width == 0 || height == 0 {
        return Err("scaled template size is too small".to_string());
    }
    Ok(resize(template, width, height, FilterType::CatmullRom))
}

fn merge_icon_matches(mut candidates: Vec<IconMatch>, max_results: usize) -> Vec<IconMatch> {
    candidates.sort_by(|a, b| {
        match b.score.partial_cmp(&a.score).unwrap_or(Ordering::Equal) {
            Ordering::Equal => item_position_order_bbox(&a.bbox, &b.bbox),
            other => other,
        }
    });

    let mut filtered = Vec::new();
    for candidate in candidates {
        if filtered
            .iter()
            .any(|kept: &IconMatch| bbox_overlap_ratio(&kept.bbox, &candidate.bbox) > 0.5)
        {
            continue;
        }
        filtered.push(candidate);
        if filtered.len() >= max_results {
            break;
        }
    }
    filtered
}

fn to_grayscale(image: &RgbaImage) -> Vec<f64> {
    image
        .pixels()
        .map(|pixel| {
            let channels = pixel.channels();
            let r = f64::from(channels[0]);
            let g = f64::from(channels[1]);
            let b = f64::from(channels[2]);
            let a = f64::from(channels[3]) / 255.0;
            (((0.299 * r) + (0.587 * g) + (0.114 * b)) * a) / 255.0
        })
        .collect()
}

fn template_similarity(
    haystack: &[f64],
    haystack_width: usize,
    needle: &[f64],
    needle_width: usize,
    needle_height: usize,
    start_x: usize,
    start_y: usize,
) -> f64 {
    let mut diff_sum = 0.0;
    let mut count = 0usize;

    for row in 0..needle_height {
        let haystack_row = (start_y + row) * haystack_width;
        let needle_row = row * needle_width;
        for col in 0..needle_width {
            let haystack_value = haystack[haystack_row + start_x + col];
            let needle_value = needle[needle_row + col];
            diff_sum += (haystack_value - needle_value).abs();
            count += 1;
        }
    }

    if count == 0 {
        return 0.0;
    }
    (1.0 - (diff_sum / count as f64)).clamp(0.0, 1.0)
}

fn bbox_overlap_ratio(a: &ScreenBoundingBox, b: &ScreenBoundingBox) -> f64 {
    let left = a.x.max(b.x);
    let top = a.y.max(b.y);
    let right = (a.x + a.width).min(b.x + b.width);
    let bottom = (a.y + a.height).min(b.y + b.height);
    let overlap_width = (right - left).max(0);
    let overlap_height = (bottom - top).max(0);
    if overlap_width == 0 || overlap_height == 0 {
        return 0.0;
    }

    let overlap_area = f64::from(overlap_width * overlap_height);
    let a_area = f64::from(a.width.max(0) * a.height.max(0));
    let b_area = f64::from(b.width.max(0) * b.height.max(0));
    overlap_area / a_area.min(b_area).max(1.0)
}

fn item_position_order(a: &OcrItem, b: &OcrItem) -> Ordering {
    item_position_order_bbox(&a.bbox, &b.bbox)
}

fn item_position_order_bbox(a: &ScreenBoundingBox, b: &ScreenBoundingBox) -> Ordering {
    match a.y.cmp(&b.y) {
        Ordering::Equal => a.x.cmp(&b.x),
        other => other,
    }
}

#[cfg(test)]
mod tests {
    use super::{
        IconMatch, ScreenBoundingBox, ScreenPoint, bbox_overlap_ratio, find_template_matches,
        item_position_order_bbox, map_tesseract_language, parse_languages, parse_tesseract_tsv,
    };
    use serde_json::json;
    use std::cmp::Ordering;
    use xcap::image::{Rgba, RgbaImage};

    #[test]
    fn parse_languages_defaults_to_chinese_and_english() {
        let languages = parse_languages(&json!({})).expect("languages should parse");
        assert_eq!(languages, vec!["chi_sim".to_string(), "eng".to_string()]);
    }

    #[test]
    fn map_tesseract_language_normalizes_common_aliases() {
        assert_eq!(map_tesseract_language("zh"), "chi_sim");
        assert_eq!(map_tesseract_language("en"), "eng");
        assert_eq!(map_tesseract_language("deu"), "deu");
    }

    #[test]
    fn parse_tesseract_tsv_filters_and_offsets_items() {
        let input = "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n\
1\t1\t0\t0\t0\t0\t0\t0\t100\t50\t-1\t\n\
5\t1\t1\t1\t1\t1\t10\t20\t40\t12\t95.5\t文件\n\
5\t1\t1\t1\t1\t2\t60\t20\t20\t12\t40.0\t忽略\n";

        let items = parse_tesseract_tsv(input, 100, 200, 0.8).expect("tsv should parse");
        assert_eq!(items.len(), 1);
        assert_eq!(items[0].text, "文件");
        assert_eq!(
            items[0].bbox,
            ScreenBoundingBox {
                x: 110,
                y: 220,
                width: 40,
                height: 12
            }
        );
        assert_eq!(items[0].center, ScreenPoint { x: 130, y: 226 });
    }

    #[test]
    fn find_template_matches_returns_ranked_deduped_candidates() {
        let mut haystack = RgbaImage::from_pixel(5, 5, Rgba([10, 10, 10, 255]));
        haystack.put_pixel(2, 1, Rgba([250, 250, 250, 255]));
        haystack.put_pixel(3, 1, Rgba([250, 250, 250, 255]));
        haystack.put_pixel(2, 2, Rgba([250, 250, 250, 255]));
        haystack.put_pixel(3, 2, Rgba([250, 250, 250, 255]));

        let template = RgbaImage::from_pixel(2, 2, Rgba([250, 250, 250, 255]));
        let matches = find_template_matches(&haystack, &template, 100, 200, 0.99, 3)
            .expect("template match should succeed");
        assert_eq!(matches.len(), 1);
        assert_eq!(
            matches[0],
            IconMatch {
                score: 1.0,
                bbox: ScreenBoundingBox {
                    x: 102,
                    y: 201,
                    width: 2,
                    height: 2
                },
                center: ScreenPoint { x: 103, y: 202 },
                scale: 1.0
            }
        );
    }

    #[test]
    fn bbox_overlap_ratio_uses_smaller_area_for_dedupe() {
        let overlap = bbox_overlap_ratio(
            &ScreenBoundingBox {
                x: 0,
                y: 0,
                width: 10,
                height: 10,
            },
            &ScreenBoundingBox {
                x: 2,
                y: 2,
                width: 6,
                height: 6,
            },
        );
        assert!(overlap > 0.9, "unexpected overlap ratio: {overlap}");
    }

    #[test]
    fn item_position_order_bbox_prefers_top_then_left() {
        let left = ScreenBoundingBox {
            x: 10,
            y: 20,
            width: 1,
            height: 1,
        };
        let right = ScreenBoundingBox {
            x: 20,
            y: 20,
            width: 1,
            height: 1,
        };
        assert_eq!(item_position_order_bbox(&left, &right), Ordering::Less);
    }
}
