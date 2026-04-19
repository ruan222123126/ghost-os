use super::params::ScaleRange;
use super::types::{IconMatch, ScreenBoundingBox, ScreenPoint};
use rayon::prelude::*;
use std::cmp::Ordering;
use xcap::image::imageops::{FilterType, resize};
use xcap::image::{Pixel, RgbaImage};

#[derive(Clone, Copy)]
struct TemplateProbe {
    row: usize,
    col: usize,
    value: u8,
    priority: u8,
}

const PERFECT_MATCH_EPSILON: f64 = 1e-6;

type MatchCandidate = (usize, usize, f64);

struct MatchSearchContext {
    haystack: Vec<u8>,
    haystack_width: usize,
    haystack_height: usize,
    template_width: usize,
    template_height: usize,
    probes: Vec<TemplateProbe>,
    max_allowed_diff: u64,
    score_denominator: f64,
}

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
    let context = build_match_search_context(image, template, threshold)?;
    if max_results == 1 {
        return find_single_template_match(&context, origin_x, origin_y);
    }
    find_multi_template_matches(&context, origin_x, origin_y, max_results)
}

fn find_single_template_match(
    context: &MatchSearchContext,
    origin_x: i32,
    origin_y: i32,
) -> Result<Vec<IconMatch>, String> {
    let max_x = context.haystack_width - context.template_width;
    let max_y = context.haystack_height - context.template_height;
    let best = (0..=max_y)
        .into_par_iter()
        .filter_map(|y| {
            let mut row_best: Option<MatchCandidate> = None;
            for x in 0..=max_x {
                let Some(score) = template_similarity_if_above_threshold(
                    &context.haystack,
                    context.haystack_width,
                    &context.probes,
                    x,
                    y,
                    context.max_allowed_diff,
                    context.score_denominator,
                ) else {
                    continue;
                };
                let candidate = (x, y, score);
                if should_replace_single_candidate(row_best.as_ref(), &candidate) {
                    row_best = Some(candidate);
                }
            }
            row_best
        })
        .reduce_with(|left, right| {
            if should_replace_single_candidate(Some(&left), &right) {
                right
            } else {
                left
            }
        });
    let Some((x, y, score)) = best else {
        return Ok(vec![]);
    };
    Ok(vec![build_icon_match(
        x,
        y,
        score,
        origin_x,
        origin_y,
        context.template_width,
        context.template_height,
    )?])
}

fn find_multi_template_matches(
    context: &MatchSearchContext,
    origin_x: i32,
    origin_y: i32,
    max_results: usize,
) -> Result<Vec<IconMatch>, String> {
    let max_x = context.haystack_width - context.template_width;
    let max_y = context.haystack_height - context.template_height;
    let mut candidates = Vec::new();
    let mut perfect_matches = Vec::new();
    for y in 0..=max_y {
        for x in 0..=max_x {
            let Some(score) = template_similarity_if_above_threshold(
                &context.haystack,
                context.haystack_width,
                &context.probes,
                x,
                y,
                context.max_allowed_diff,
                context.score_denominator,
            ) else {
                continue;
            };
            let candidate = build_icon_match(
                x,
                y,
                score,
                origin_x,
                origin_y,
                context.template_width,
                context.template_height,
            )?;
            if is_perfect_match(score)
                && append_non_overlapping_match(&mut perfect_matches, &candidate)
                && perfect_matches.len() >= max_results
            {
                return Ok(perfect_matches);
            }
            candidates.push(candidate);
        }
    }
    Ok(merge_icon_matches(candidates, max_results))
}

fn build_match_search_context(
    image: &RgbaImage,
    template: &RgbaImage,
    threshold: f64,
) -> Result<MatchSearchContext, String> {
    if template.width() == 0 || template.height() == 0 {
        return Err("template image must not be empty".to_string());
    }
    if template.width() > image.width() || template.height() > image.height() {
        return Err("template image is larger than search region".to_string());
    }
    let template_width =
        usize::try_from(template.width()).map_err(|_| "template width is too large".to_string())?;
    let template_height = usize::try_from(template.height())
        .map_err(|_| "template height is too large".to_string())?;
    let haystack_width =
        usize::try_from(image.width()).map_err(|_| "image width is too large".to_string())?;
    let haystack_height =
        usize::try_from(image.height()).map_err(|_| "image height is too large".to_string())?;
    let haystack = to_grayscale(image);
    let needle = to_grayscale(template);
    let probes = build_template_probes(&needle, template_width);
    let pixel_count = template_width * template_height;
    let max_allowed_diff =
        ((1.0 - threshold).clamp(0.0, 1.0) * 255.0 * pixel_count as f64).ceil() as u64;
    Ok(MatchSearchContext {
        haystack,
        haystack_width,
        haystack_height,
        template_width,
        template_height,
        probes,
        max_allowed_diff,
        score_denominator: 255.0 * pixel_count as f64,
    })
}

fn build_icon_match(
    x: usize,
    y: usize,
    score: f64,
    origin_x: i32,
    origin_y: i32,
    template_width: usize,
    template_height: usize,
) -> Result<IconMatch, String> {
    let x_i32 = i32::try_from(x).map_err(|_| "match x is too large".to_string())?;
    let y_i32 = i32::try_from(y).map_err(|_| "match y is too large".to_string())?;
    let width_i32 =
        i32::try_from(template_width).map_err(|_| "template width is too large".to_string())?;
    let height_i32 =
        i32::try_from(template_height).map_err(|_| "template height is too large".to_string())?;
    let abs_x = origin_x + x_i32;
    let abs_y = origin_y + y_i32;
    Ok(IconMatch {
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
    })
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

fn to_grayscale(image: &RgbaImage) -> Vec<u8> {
    image
        .pixels()
        .map(|pixel| {
            let channels = pixel.channels();
            let r = u32::from(channels[0]);
            let g = u32::from(channels[1]);
            let b = u32::from(channels[2]);
            let a = u32::from(channels[3]);
            let luminance = ((299 * r) + (587 * g) + (114 * b) + 500) / 1000;
            ((luminance * a + 127) / 255) as u8
        })
        .collect()
}

fn build_template_probes(needle: &[u8], template_width: usize) -> Vec<TemplateProbe> {
    let mean = if needle.is_empty() {
        0
    } else {
        let total: u64 = needle.iter().map(|value| u64::from(*value)).sum();
        (total as f64 / needle.len() as f64).round() as u8
    };
    let mut probes = Vec::with_capacity(needle.len());
    for (index, value) in needle.iter().enumerate() {
        probes.push(TemplateProbe {
            row: index / template_width,
            col: index % template_width,
            value: *value,
            priority: value.abs_diff(mean),
        });
    }
    probes.sort_by(|left, right| match right.priority.cmp(&left.priority) {
        Ordering::Equal => match left.row.cmp(&right.row) {
            Ordering::Equal => left.col.cmp(&right.col),
            other => other,
        },
        other => other,
    });
    probes
}

fn template_similarity_if_above_threshold(
    haystack: &[u8],
    haystack_width: usize,
    probes: &[TemplateProbe],
    start_x: usize,
    start_y: usize,
    max_allowed_diff: u64,
    score_denominator: f64,
) -> Option<f64> {
    let mut diff_sum = 0u64;

    for probe in probes {
        let index = (start_y + probe.row) * haystack_width + start_x + probe.col;
        diff_sum += u64::from(haystack[index].abs_diff(probe.value));
        if diff_sum > max_allowed_diff {
            return None;
        }
    }
    Some((1.0 - (diff_sum as f64 / score_denominator)).clamp(0.0, 1.0))
}

fn is_perfect_match(score: f64) -> bool {
    (1.0 - score).abs() <= PERFECT_MATCH_EPSILON
}

fn append_non_overlapping_match(matches: &mut Vec<IconMatch>, candidate: &IconMatch) -> bool {
    if matches
        .iter()
        .any(|item| bbox_overlap_ratio(&item.bbox, &candidate.bbox) > 0.5)
    {
        return false;
    }
    matches.push(candidate.clone());
    true
}

fn should_replace_single_candidate(
    current: Option<&MatchCandidate>,
    candidate: &MatchCandidate,
) -> bool {
    let Some(current) = current else {
        return true;
    };
    let score_cmp = candidate
        .2
        .partial_cmp(&current.2)
        .unwrap_or(Ordering::Equal);
    match score_cmp {
        Ordering::Greater => true,
        Ordering::Equal => {
            candidate.1 < current.1 || (candidate.1 == current.1 && candidate.0 < current.0)
        }
        Ordering::Less => false,
    }
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
