use pyo3::prelude::*;
use reqwest::Url;
use serde_json::json;

use super::SandboxConfig;
use super::tool_runtime::ToolRuntime;
use super::web_security::{fetch_and_convert_webpage, validate_https_url};

pub(crate) fn fetch_webpage_py(
    runtime: &ToolRuntime,
    py: Python<'_>,
    url: String,
) -> PyResult<String> {
    let url = url.trim().to_string();
    let args = json!({ "url": &url });
    if let Err(err) = runtime.consume_web_request_budget() {
        return Err(runtime.log_error("fetch_webpage", args.clone(), err));
    }
    let parsed = match validate_https_url(&url) {
        Ok(parsed) => parsed,
        Err(err) => return Err(runtime.log_error("fetch_webpage", args, err)),
    };

    match py.allow_threads(|| fetch_webpage_impl(runtime.config(), &parsed)) {
        Ok(markdown) => {
            runtime.log_success("fetch_webpage", args, markdown.clone());
            Ok(markdown)
        }
        Err(err) => Err(runtime.log_error("fetch_webpage", args, err)),
    }
}

pub(crate) fn fetch_webpage_impl(config: &SandboxConfig, url: &Url) -> Result<String, String> {
    let markdown = fetch_and_convert_webpage(url, config.max_webpage_bytes)?;
    Ok(truncate_to_bytes(&markdown, config.max_webpage_bytes))
}

fn truncate_to_bytes(text: &str, max_bytes: usize) -> String {
    if text.len() <= max_bytes {
        return text.to_string();
    }

    let mut end = max_bytes;
    while !text.is_char_boundary(end) {
        end = end.saturating_sub(1);
        if end == 0 {
            return String::new();
        }
    }
    text[..end].to_string()
}
