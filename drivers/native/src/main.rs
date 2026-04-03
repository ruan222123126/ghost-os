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

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum EntryRoute {
    SandboxWorker,
    Persistent,
    OneShot,
}

#[derive(Debug, Eq, PartialEq)]
struct RouteResult {
    handled: bool,
    error: Option<String>,
}

impl RouteResult {
    fn handled() -> Self {
        Self {
            handled: true,
            error: None,
        }
    }

    fn error(message: impl Into<String>) -> Self {
        Self {
            handled: false,
            error: Some(message.into()),
        }
    }
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let result = route_entry(&args);
    if let Some(err) = result.error {
        eprintln!("{err}");
        std::process::exit(1);
    }
    if !result.handled {
        eprintln!("native entry route was not handled");
        std::process::exit(1);
    }
}

fn route_entry(args: &[String]) -> RouteResult {
    let route = match resolve_entry_route(args) {
        Ok(route) => route,
        Err(err) => return RouteResult::error(err),
    };

    match route {
        EntryRoute::SandboxWorker => {
            script_exec::run_sandbox_worker();
            RouteResult::handled()
        }
        EntryRoute::Persistent => match run_persistent_mode() {
            Ok(()) => RouteResult::handled(),
            Err(err) => RouteResult::error(format!("persistent mode exited: {err}")),
        },
        EntryRoute::OneShot => run_oneshot_mode(),
    }
}

fn resolve_entry_route(args: &[String]) -> Result<EntryRoute, String> {
    let mut sandbox_worker = false;
    let mut persistent = false;

    for arg in args.iter().skip(1) {
        match arg.as_str() {
            "--sandbox-worker" => sandbox_worker = true,
            "--persistent" => persistent = true,
            _ => return Err(format!("unknown argument: {arg}")),
        }
    }

    if sandbox_worker && persistent {
        return Err(
            "conflicting arguments: --sandbox-worker and --persistent cannot be used together"
                .to_string(),
        );
    }

    if sandbox_worker {
        return Ok(EntryRoute::SandboxWorker);
    }
    if persistent {
        return Ok(EntryRoute::Persistent);
    }
    Ok(EntryRoute::OneShot)
}

fn run_oneshot_mode() -> RouteResult {
    let response = match build_oneshot_response() {
        Ok(response) => response,
        Err(err) => Response::error(err),
    };

    match emit(response) {
        Ok(()) => RouteResult::handled(),
        Err(err) => RouteResult::error(format!("oneshot mode emit failed: {err}")),
    }
}

fn build_oneshot_response() -> Result<Response, String> {
    let input = read_stdin_payload()?;
    let request: Request =
        serde_json::from_str(&input).map_err(|err| format!("invalid json: {err}"))?;
    Ok(
        action_router::dispatch_action(&request.action, &request.params, &request.trace_id)
            .with_request_id(request.request_id.as_deref()),
    )
}

fn run_persistent_mode() -> Result<(), String> {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut reader = BufReader::new(stdin.lock());
    let mut writer = BufWriter::new(stdout.lock());
    serve_persistent_session(&mut reader, &mut writer)
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

fn emit(response: Response) -> Result<(), String> {
    let stdout = io::stdout();
    let mut writer = stdout.lock();
    write_response_line(&mut writer, &response)
}

fn write_response_line<W: Write>(writer: &mut W, response: &Response) -> Result<(), String> {
    let mut bytes =
        serde_json::to_vec(response).map_err(|err| format!("encode response failed: {err}"))?;
    bytes.push(b'\n');
    writer
        .write_all(&bytes)
        .map_err(|err| format!("write response failed: {err}"))?;
    writer
        .flush()
        .map_err(|err| format!("flush response failed: {err}"))?;
    Ok(())
}

#[cfg(test)]
mod main_startup_tests;

#[cfg(test)]
mod tests {
    use super::{Response, serve_persistent_session, write_response_line};
    use serde_json::json;
    use std::io::{Cursor, Error, Write};

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

    #[test]
    fn write_response_line_returns_error_when_writer_fails() {
        let mut writer = BrokenWriter;
        let err = write_response_line(&mut writer, &Response::success(json!({})))
            .expect_err("write failure should be surfaced");
        assert!(err.contains("write response failed"));
    }

    struct BrokenWriter;

    impl Write for BrokenWriter {
        fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
            Err(Error::other("broken pipe"))
        }

        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
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
