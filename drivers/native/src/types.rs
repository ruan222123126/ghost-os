use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

// Request 对齐 core/shared/schema.json 的请求结构。
#[derive(Debug, Deserialize)]
pub(crate) struct Request {
    pub(crate) action: String,
    #[serde(rename = "params", default)]
    pub(crate) params: Value,
    #[serde(rename = "trace_id", default)]
    pub(crate) trace_id: String,
    #[serde(rename = "request_id", default)]
    pub(crate) request_id: Option<String>,
}

// Response 是 native 层统一返回格式。
#[derive(Debug, Serialize, Deserialize)]
pub(crate) struct Response {
    pub(crate) status: String,
    pub(crate) payload: Value,
    pub(crate) error: String,
    #[serde(rename = "request_id", skip_serializing_if = "Option::is_none")]
    pub(crate) request_id: Option<String>,
}

impl Response {
    // success 构造成功响应。
    pub(crate) fn success(payload: Value) -> Self {
        Self {
            status: "success".to_string(),
            payload,
            error: String::new(),
            request_id: None,
        }
    }

    // error 构造失败响应。
    pub(crate) fn error(message: String) -> Self {
        Self {
            status: "error".to_string(),
            payload: json!({}),
            error: message,
            request_id: None,
        }
    }

    // with_request_id 为响应附加 request_id，便于 persistent 模式做一一对应。
    pub(crate) fn with_request_id(mut self, request_id: Option<&str>) -> Self {
        self.request_id = request_id
            .map(str::trim)
            .filter(|value| !value.is_empty())
            .map(ToOwned::to_owned);
        self
    }
}
