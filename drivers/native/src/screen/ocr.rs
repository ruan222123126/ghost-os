use super::image_ops::decode_png_base64;
use super::types::{OcrItem, ScreenBoundingBox, ScreenPoint};
use crate::Response;
use crate::json_params::{optional_f64, optional_i32, required_string};
use serde_json::{Value, json};
use std::cmp::Ordering;
use std::fs;
use std::path::Path;
use std::process::Command;
use std::time::{SystemTime, UNIX_EPOCH};
use xcap::image::{DynamicImage, RgbaImage};

pub(crate) fn handle_ocr_image(params: &Value) -> Response {
    let image_base64 = match required_string(params, "image_base64") {
        Ok(value) => value,
        Err(err) => return Response::error(err),
    };
    let image = match decode_png_base64(&image_base64) {
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
    let languages = match parse_languages(params) {
        Ok(languages) => languages,
        Err(err) => return Response::error(err),
    };
    let min_confidence = match optional_f64(params, "min_confidence") {
        Ok(Some(value)) => value.clamp(0.0, 1.0),
        Ok(None) => 0.75,
        Err(err) => return Response::error(err),
    };
    let output = match run_tesseract_ocr(&image, &languages) {
        Ok(output) => output,
        Err(err) => return Response::error(err),
    };
    let mut items = match parse_tesseract_tsv(&output, origin_x, origin_y, min_confidence) {
        Ok(items) => items,
        Err(err) => return Response::error(err),
    };
    items.sort_by(item_position_order);
    Response::success(json!({ "items": items }))
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
    origin_x: i32,
    origin_y: i32,
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

        let left = parse_i32(cols[6], "left")?;
        let top = parse_i32(cols[7], "top")?;
        let width = parse_i32(cols[8], "width")?;
        let height = parse_i32(cols[9], "height")?;
        if width <= 0 || height <= 0 {
            continue;
        }

        let x = origin_x + left;
        let y = origin_y + top;
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

fn parse_i32(raw: &str, field: &str) -> Result<i32, String> {
    raw.trim()
        .parse::<i32>()
        .map_err(|err| format!("decode tesseract {field} failed: {err}"))
}

fn item_position_order(a: &OcrItem, b: &OcrItem) -> Ordering {
    match a.bbox.y.cmp(&b.bbox.y) {
        Ordering::Equal => a.bbox.x.cmp(&b.bbox.x),
        other => other,
    }
}

#[cfg(test)]
mod tests {
    use super::{map_tesseract_language, parse_languages, parse_tesseract_tsv};
    use crate::screen::types::{ScreenBoundingBox, ScreenPoint};
    use serde_json::json;

    #[test]
    fn parse_languages_defaults_to_chinese_and_english() {
        let languages = parse_languages(&json!({})).expect("must parse");
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

        let items = parse_tesseract_tsv(input, 100, 200, 0.8).expect("must parse");
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
}
