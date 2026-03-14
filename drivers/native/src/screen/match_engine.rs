use super::params::ScaleRange;
use super::types::{IconMatch, ScreenBoundingBox, ScreenPoint};
use std::cmp::Ordering;
use xcap::image::imageops::{FilterType, resize};
use xcap::image::{Pixel, RgbaImage};

pub(crate) fn find_template_matches_with_scale(
    image: &RgbaImage,
    template: &RgbaImage,
    origin_x: i32,
    origin_y: i32,
    threshold: f64,
    max_results: usize,
    scale_range: Option<ScaleRange>,
) -> Result<Vec<IconMatch>, String> {
    let Some(range) = scale_range else {
        return find_template_matches(image, template, origin_x, origin_y, threshold, max_results);
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
        let mut matches =
            find_template_matches(image, &scaled, origin_x, origin_y, threshold, per_scale_max)?;
        for item in &mut matches {
            item.scale = scale;
        }
        candidates.extend(matches);
    }
    Ok(merge_icon_matches(candidates, max_results))
}

fn find_template_matches(
    image: &RgbaImage,
    template: &RgbaImage,
    origin_x: i32,
    origin_y: i32,
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
            let abs_x = origin_x + x_i32;
            let abs_y = origin_y + y_i32;
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
    Ok(merge_icon_matches(candidates, max_results))
}

fn build_scale_steps(range: ScaleRange) -> Result<Vec<f64>, String> {
    let mut scales = Vec::new();
    let mut value = range.min;
    let mut steps = 0usize;
    while value <= range.max + 1e-6 {
        scales.push(value);
        value += range.step;
        steps += 1;
        if steps > 15 {
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

fn item_position_order_bbox(a: &ScreenBoundingBox, b: &ScreenBoundingBox) -> Ordering {
    match a.y.cmp(&b.y) {
        Ordering::Equal => a.x.cmp(&b.x),
        other => other,
    }
}

#[cfg(test)]
mod tests;
