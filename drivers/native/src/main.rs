#![allow(unsafe_op_in_unsafe_fn)]

mod action_router;
mod codex_cli;
mod display_scale;
mod file_actions;
mod framing;
mod input;
mod json_params;
mod sandbox;
mod screen;
mod script_exec;
mod shell_actions;
mod types;

use std::io::{self, BufReader, BufWriter, Read, Write};
use types::Request;

pub(crate) use types::Response;

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.iter().any(|arg| arg == "--sandbox-worker") {
        script_exec::run_sandbox_worker();
        return;
    }

    if args.iter().any(|arg| arg == "--persistent") {
        run_persistent_mode();
        return;
    }

    run_oneshot_mode();
}

fn run_oneshot_mode() {
    // 读取整段 stdin，保持最小协议处理路径。
    let input = match read_stdin_payload() {
        Ok(input) => input,
        Err(err) => {
            emit(Response::error(err));
            return;
        }
    };

    let request: Request = match serde_json::from_str(&input) {
        Ok(request) => request,
        Err(err) => {
            emit(Response::error(format!("invalid json: {err}")));
            return;
        }
    };

    let response =
        action_router::dispatch_action(&request.action, &request.params, &request.trace_id)
            .with_request_id(request.request_id.as_deref());
    emit(response);
}

fn run_persistent_mode() {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut reader = BufReader::new(stdin.lock());
    let mut writer = BufWriter::new(stdout.lock());

    if let Err(err) = serve_persistent_session(&mut reader, &mut writer) {
        eprintln!("persistent mode exited: {err}");
    }
}

fn serve_persistent_session<R: Read, W: Write>(
    reader: &mut R,
    writer: &mut W,
) -> Result<(), String> {
    let mut state = action_router::PersistentState::new();

    loop {
        let request = match framing::read_frame(reader) {
            Ok(request) => request,
            Err(framing::ReadFrameError::Closed) => return Ok(()),
            Err(framing::ReadFrameError::Request {
                message,
                request_id,
            }) => {
                let response = Response::error(message).with_request_id(request_id.as_deref());
                framing::write_frame(writer, &response)?;
                continue;
            }
            Err(framing::ReadFrameError::Protocol(message)) => return Err(message),
        };

        let response = action_router::dispatch_with_state(
            &request.action,
            &request.params,
            &request.trace_id,
            &mut state,
        )
        .with_request_id(request.request_id.as_deref());

        framing::write_frame(writer, &response)?;
    }
}

pub(crate) fn read_stdin_payload() -> Result<String, String> {
    let mut input = String::new();
    io::stdin()
        .read_to_string(&mut input)
        .map_err(|_| "failed to read stdin".to_string())?;

    let input = input.trim();
    if input.is_empty() {
        return Err("empty input".to_string());
    }

    Ok(input.to_string())
}

// emit 负责写出单行 JSON 响应，失败时静默返回。
fn emit(response: Response) {
    let mut stdout = io::stdout();
    if let Ok(mut bytes) = serde_json::to_vec(&response) {
        bytes.push(b'\n');
        let _ = stdout.write_all(&bytes);
        let _ = stdout.flush();
    }
}

#[cfg(test)]
mod tests {
    use super::{Response, serve_persistent_session};
    use serde_json::json;
    use std::io::Cursor;

    #[test]
    fn serve_persistent_session_handles_multiple_requests() {
        let mut input = Vec::new();
        input.extend(frame_request(&json!({
            "action": "PING",
            "params": {},
            "trace_id": "trace-a",
            "request_id": "req-a"
        })));
        input.extend(frame_request(&json!({
            "action": "PING",
            "params": {},
            "trace_id": "trace-b",
            "request_id": "req-b"
        })));

        let mut reader = Cursor::new(input);
        let mut writer = Vec::new();
        serve_persistent_session(&mut reader, &mut writer).expect("session should complete");

        let mut output = Cursor::new(writer);
        let first = read_response_frame(&mut output);
        let second = read_response_frame(&mut output);

        assert_eq!(first.request_id.as_deref(), Some("req-a"));
        assert_eq!(first.payload["message"], "PONG");
        assert_eq!(second.request_id.as_deref(), Some("req-b"));
        assert_eq!(second.payload["trace_id"], "trace-b");
    }

    fn frame_request(payload: &serde_json::Value) -> Vec<u8> {
        let encoded = serde_json::to_vec(payload).expect("request should encode");
        let mut frame = Vec::with_capacity(4 + encoded.len());
        frame.extend_from_slice(&(encoded.len() as u32).to_be_bytes());
        frame.extend_from_slice(&encoded);
        frame
    }

    fn read_response_frame(reader: &mut Cursor<Vec<u8>>) -> Response {
        let mut length_buf = [0u8; 4];
        std::io::Read::read_exact(reader, &mut length_buf).expect("length should decode");
        let length = u32::from_be_bytes(length_buf) as usize;
        let mut payload = vec![0u8; length];
        std::io::Read::read_exact(reader, &mut payload).expect("payload should decode");
        serde_json::from_slice(&payload).expect("response should decode")
    }
}
