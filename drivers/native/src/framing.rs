use serde_json::Value;
use std::fmt;
use std::io::{self, Read, Write};

use crate::types::{Request, Response};

pub(crate) const MAX_FRAME_BYTES: usize = 16 * 1024 * 1024;

#[derive(Debug)]
pub(crate) enum ReadFrameError {
    Closed,
    Request {
        message: String,
        request_id: Option<String>,
    },
    Protocol(String),
}

impl fmt::Display for ReadFrameError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Closed => write!(f, "input closed"),
            Self::Request { message, .. } => write!(f, "{message}"),
            Self::Protocol(message) => write!(f, "{message}"),
        }
    }
}

pub(crate) fn read_frame<R: Read>(reader: &mut R) -> Result<Request, ReadFrameError> {
    let mut length_buf = [0u8; 4];
    match reader.read_exact(&mut length_buf) {
        Ok(()) => {}
        Err(err) if err.kind() == io::ErrorKind::UnexpectedEof => {
            return Err(ReadFrameError::Closed);
        }
        Err(err) => {
            return Err(ReadFrameError::Protocol(format!(
                "read frame length failed: {err}"
            )));
        }
    }

    let length = u32::from_be_bytes(length_buf) as usize;
    if length == 0 || length > MAX_FRAME_BYTES {
        return Err(ReadFrameError::Protocol(format!(
            "invalid frame length: {length}"
        )));
    }

    let mut payload = vec![0u8; length];
    reader
        .read_exact(&mut payload)
        .map_err(|err| ReadFrameError::Protocol(format!("read frame payload failed: {err}")))?;

    let value: Value = serde_json::from_slice(&payload).map_err(|err| ReadFrameError::Request {
        message: format!("invalid json: {err}"),
        request_id: None,
    })?;
    let request_id = extract_request_id(&value);

    serde_json::from_value(value).map_err(|err| ReadFrameError::Request {
        message: format!("invalid request: {err}"),
        request_id,
    })
}

pub(crate) fn write_frame<W: Write>(writer: &mut W, response: &Response) -> Result<(), String> {
    let payload =
        serde_json::to_vec(response).map_err(|err| format!("serialize frame failed: {err}"))?;
    if payload.is_empty() || payload.len() > MAX_FRAME_BYTES {
        return Err(format!("response frame is too large: {}", payload.len()));
    }

    writer
        .write_all(&(payload.len() as u32).to_be_bytes())
        .map_err(|err| format!("write frame length failed: {err}"))?;
    writer
        .write_all(&payload)
        .map_err(|err| format!("write frame payload failed: {err}"))?;
    writer
        .flush()
        .map_err(|err| format!("flush frame failed: {err}"))?;
    Ok(())
}

fn extract_request_id(value: &Value) -> Option<String> {
    value
        .get("request_id")
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|request_id| !request_id.is_empty())
        .map(ToOwned::to_owned)
}

#[cfg(test)]
mod tests {
    use super::{MAX_FRAME_BYTES, ReadFrameError, read_frame, write_frame};
    use crate::types::{Request, Response};
    use serde_json::json;
    use std::io::Cursor;

    #[test]
    fn read_frame_accepts_valid_payload() {
        let payload = json!({
            "action": "PING",
            "params": {},
            "trace_id": "trace-1",
            "request_id": "req-1"
        });
        let bytes = encode_payload(&payload);

        let mut reader = Cursor::new(bytes);
        let request = read_frame(&mut reader).expect("frame should decode");

        assert_eq!(request.action, "PING");
        assert_eq!(request.trace_id, "trace-1");
        assert_eq!(request.request_id.as_deref(), Some("req-1"));
    }

    #[test]
    fn read_frame_rejects_invalid_length() {
        let mut bytes = Vec::new();
        bytes.extend_from_slice(&((MAX_FRAME_BYTES + 1) as u32).to_be_bytes());
        bytes.extend_from_slice(br#"{}"#);

        let mut reader = Cursor::new(bytes);
        let err = read_frame(&mut reader).expect_err("invalid length must fail");

        match err {
            ReadFrameError::Protocol(message) => {
                assert!(message.contains("invalid frame length"));
            }
            other => panic!("unexpected error: {other}"),
        }
    }

    #[test]
    fn read_frame_surfaces_json_decode_error() {
        let mut frame = Vec::new();
        frame.extend_from_slice(&(4_u32).to_be_bytes());
        frame.extend_from_slice(b"{bad");
        let mut reader = Cursor::new(frame);
        let err = read_frame(&mut reader).expect_err("must fail");
        match err {
            ReadFrameError::Request {
                message,
                request_id,
            } => {
                assert!(message.contains("invalid json"));
                assert_eq!(request_id, None);
            }
            other => panic!("unexpected error: {other}"),
        }
    }

    #[test]
    fn read_frame_preserves_request_id_when_schema_invalid() {
        let payload = json!({
            "action": 123,
            "params": {},
            "request_id": " req-99 "
        });
        let bytes = encode_payload(&payload);
        let mut reader = Cursor::new(bytes);
        let err = read_frame(&mut reader).expect_err("must fail");
        match err {
            ReadFrameError::Request {
                message,
                request_id,
            } => {
                assert!(message.contains("invalid request"));
                assert_eq!(request_id, Some("req-99".to_string()));
            }
            other => panic!("unexpected error: {other}"),
        }
    }

    #[test]
    fn read_frame_surfaces_payload_truncation_as_protocol_error() {
        let payload = br#"{"action":"PING"}"#;
        let mut frame = Vec::new();
        frame.extend_from_slice(&((payload.len() as u32 + 5).to_be_bytes()));
        frame.extend_from_slice(payload);
        let mut reader = Cursor::new(frame);
        let err = read_frame(&mut reader).expect_err("must fail");
        match err {
            ReadFrameError::Protocol(message) => {
                assert!(message.contains("read frame payload failed"));
            }
            other => panic!("unexpected error: {other}"),
        }
    }

    #[test]
    fn write_frame_roundtrip_preserves_response() {
        let response = Response::success(json!({
            "message": "PONG"
        }))
        .with_request_id(Some("req-2"));

        let mut bytes = Vec::new();
        write_frame(&mut bytes, &response).expect("frame should encode");

        let request_like = decode_response_frame(&bytes).expect("frame should decode");
        assert_eq!(request_like.status, "success");
        assert_eq!(request_like.request_id.as_deref(), Some("req-2"));
        assert_eq!(request_like.payload["message"], "PONG");
    }

    fn encode_payload(payload: &serde_json::Value) -> Vec<u8> {
        let encoded = serde_json::to_vec(payload).expect("payload should encode");
        let mut frame = Vec::with_capacity(4 + encoded.len());
        frame.extend_from_slice(&(encoded.len() as u32).to_be_bytes());
        frame.extend_from_slice(&encoded);
        frame
    }

    fn decode_response_frame(frame: &[u8]) -> Result<Response, String> {
        if frame.len() < 4 {
            return Err("frame is too short".to_string());
        }

        let payload_len = u32::from_be_bytes([frame[0], frame[1], frame[2], frame[3]]) as usize;
        let payload = frame
            .get(4..4 + payload_len)
            .ok_or_else(|| "payload is truncated".to_string())?;

        serde_json::from_slice(payload).map_err(|err| format!("decode response failed: {err}"))
    }

    #[allow(dead_code)]
    fn _assert_request_type(_: Request) {}
}
