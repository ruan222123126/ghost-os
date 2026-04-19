use super::types::{OcrItem, ScreenBoundingBox, ScreenPoint};
use crate::Response;
use crate::json_params::{optional_f64, optional_i32, required_string};
use serde_json::{Value, json};
use std::cmp::Ordering;
use std::path::Path;
use std::process::Command;

pub(crate) fn handle_ocr_image(params: &Value) -> Response {
    let image_path = match required_string(params, "image_path") {
        Ok(path) => path,
        Err(err) => return Response::error(err),
    };
    let image_path = Path::new(&image_path);
    if !image_path.exists() {
        return Response::error(format!(
            "image_path does not exist: {}",
            image_path.display()
        ));
    }
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
    let output = match run_tesseract_ocr(image_path, &languages) {
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

fn run_tesseract_ocr(image_path: &Path, languages: &[String]) -> Result<String, String> {
    let language_arg = if languages.is_empty() {
        "eng".to_string()
    } else {
        languages.join("+")
    };
    let output = Command::new("tesseract")
        .args([
            image_path.to_string_lossy().as_ref(),
            "stdout",
            "-l",
            &language_arg,
            "--psm",
            "11",
            "tsv",
        ])
        .output();

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
    use super::{
        item_position_order, map_tesseract_language, parse_languages, parse_tesseract_tsv,
    };
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
    fn parse_languages_deduplicates_and_sorts() {
        let languages = parse_languages(&json!({
            "languages": ["EN", "zh", "eng", "zh-cn"]
        }))
        .expect("must parse");
        assert_eq!(languages, vec!["chi_sim".to_string(), "eng".to_string()]);
    }

    #[test]
    fn parse_languages_rejects_non_string_items() {
        let err = parse_languages(&json!({
            "languages": ["en", 1]
        }))
        .expect_err("must fail");
        assert!(err.contains("non-empty strings"));
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

    #[test]
    fn parse_tesseract_tsv_rejects_invalid_numeric_fields() {
        let input = "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n\
5\t1\t1\t1\t1\t1\tnan\t20\t40\t12\t95.5\t文件\n";
        let err = parse_tesseract_tsv(input, 0, 0, 0.1).expect_err("must fail");
        assert!(err.contains("decode tesseract left failed"));
    }

    #[test]
    fn item_position_order_prefers_top_then_left() {
        let mut items = vec![
            crate::screen::types::OcrItem {
                text: "B".to_string(),
                confidence: 1.0,
                bbox: ScreenBoundingBox {
                    x: 20,
                    y: 10,
                    width: 5,
                    height: 5,
                },
                center: ScreenPoint { x: 22, y: 12 },
            },
            crate::screen::types::OcrItem {
                text: "A".to_string(),
                confidence: 1.0,
                bbox: ScreenBoundingBox {
                    x: 10,
                    y: 10,
                    width: 5,
                    height: 5,
                },
                center: ScreenPoint { x: 12, y: 12 },
            },
            crate::screen::types::OcrItem {
                text: "C".to_string(),
                confidence: 1.0,
                bbox: ScreenBoundingBox {
                    x: 5,
                    y: 11,
                    width: 5,
                    height: 5,
                },
                center: ScreenPoint { x: 7, y: 13 },
            },
        ];
        items.sort_by(item_position_order);
        let ordered: Vec<String> = items.into_iter().map(|item| item.text).collect();
        assert_eq!(ordered, vec!["A", "B", "C"]);
    }
}
