use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::io::{self, Read, Write};

#[derive(Deserialize)]
struct Request {
    action: String,
    #[serde(rename = "params", default)]
    _params: Value,
    #[serde(rename = "trace_id", default)]
    trace_id: String,
}

#[derive(Serialize)]
struct Response {
    status: String,
    payload: Value,
    error: String,
}

impl Response {
    fn success(payload: Value) -> Self {
        Self {
            status: "success".to_string(),
            payload,
            error: String::new(),
        }
    }

    fn error(message: String) -> Self {
        Self {
            status: "error".to_string(),
            payload: json!({}),
            error: message,
        }
    }
}

fn main() {
    let mut input = String::new();
    if io::stdin().read_to_string(&mut input).is_err() {
        emit(Response::error("failed to read stdin".to_string()));
        return;
    }

    let input = input.trim();
    if input.is_empty() {
        emit(Response::error("empty input".to_string()));
        return;
    }

    let request: Request = match serde_json::from_str(input) {
        Ok(request) => request,
        Err(err) => {
            emit(Response::error(format!("invalid json: {err}")));
            return;
        }
    };

    if request.action == "PING" {
        emit(Response::success(json!({
            "message": "PONG",
            "trace_id": request.trace_id
        })));
        return;
    }

    emit(Response::error(format!(
        "unsupported action: {}",
        request.action
    )));
}

fn emit(response: Response) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&response) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}
