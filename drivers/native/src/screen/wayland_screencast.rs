use super::wayland_gstreamer::capture_png_from_pipewire;
use super::wayland_portal_response::PortalRequestResponse;
use dbus::Path;
use dbus::arg::{AppendAll, ArgType, OwnedFd, PropMap, RefArg, Variant};
use dbus::blocking::{Proxy, SyncConnection};
use dbus::message::{MatchRule, SignalArgs};
use std::os::fd::AsRawFd;
use std::sync::{Arc, Mutex, OnceLock};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use xcap::image::RgbaImage;

const PORTAL_DEST: &str = "org.freedesktop.portal.Desktop";
const PORTAL_PATH: &str = "/org/freedesktop/portal/desktop";
const SCREENCAST_IFACE: &str = "org.freedesktop.portal.ScreenCast";
const DBUS_CALL_TIMEOUT: Duration = Duration::from_secs(10);
const REQUEST_WAIT_TIMEOUT: Duration = Duration::from_secs(20);
const SCREENCAST_SOURCE_MONITOR: u32 = 1;

static WAYLAND_SESSION: OnceLock<Mutex<Option<WaylandSession>>> = OnceLock::new();

#[derive(Debug)]
pub(crate) struct WaylandCaptureFrame {
    pub(crate) logical_size: Option<(i32, i32)>,
    pub(crate) origin: Option<(i32, i32)>,
    pub(crate) image: RgbaImage,
}

#[derive(Clone, Copy, Debug, Default)]
struct StreamInfo {
    node_id: u32,
    position: Option<(i32, i32)>,
    size: Option<(i32, i32)>,
}

struct WaylandSession {
    _conn: SyncConnection,
    _session_handle: Path<'static>,
    stream: StreamInfo,
    pipewire_fd: OwnedFd,
}

pub(crate) fn capture_frame() -> Result<WaylandCaptureFrame, String> {
    let store = WAYLAND_SESSION.get_or_init(|| Mutex::new(None));
    let mut slot = store
        .lock()
        .map_err(|_| "wayland screencast lock is poisoned".to_string())?;
    if slot.is_none() {
        *slot = Some(establish_session()?);
    }
    let first_attempt = slot
        .as_mut()
        .ok_or_else(|| "wayland screencast session is unavailable".to_string())?
        .capture_frame();
    if let Ok(frame) = first_attempt {
        return Ok(frame);
    }

    let first_err = first_attempt
        .err()
        .unwrap_or_else(|| "unknown error".to_string());
    *slot = None;
    *slot = Some(establish_session()?);
    slot.as_mut()
        .ok_or_else(|| "wayland screencast session re-create failed".to_string())?
        .capture_frame()
        .map_err(|retry_err| format!("capture retry failed: first={first_err}; retry={retry_err}"))
}

impl WaylandSession {
    fn capture_frame(&mut self) -> Result<WaylandCaptureFrame, String> {
        let image = capture_png_from_pipewire(self.pipewire_fd.as_raw_fd(), self.stream.node_id)?;
        Ok(WaylandCaptureFrame {
            logical_size: self.stream.size,
            origin: self.stream.position,
            image,
        })
    }
}

fn establish_session() -> Result<WaylandSession, String> {
    let conn = SyncConnection::new_session()
        .map_err(|err| format!("connect session bus for screencast failed: {err}"))?;
    let session_handle = create_portal_session(&conn)?;
    select_monitor_source(&conn, &session_handle)?;
    let start_results = start_session(&conn, &session_handle)?;
    let stream = parse_primary_stream(&start_results)?;
    let pipewire_fd = open_pipewire_remote(&conn, &session_handle)?;
    Ok(WaylandSession {
        _conn: conn,
        _session_handle: session_handle,
        stream,
        pipewire_fd,
    })
}

fn create_portal_session(conn: &SyncConnection) -> Result<Path<'static>, String> {
    let mut options = PropMap::new();
    options.insert(
        "handle_token".to_string(),
        Variant(Box::new(new_token("create"))),
    );
    options.insert(
        "session_handle_token".to_string(),
        Variant(Box::new(new_token("session"))),
    );
    let results = call_portal_request(conn, "CreateSession", (options,))?;
    let handle_text = results
        .get("session_handle")
        .and_then(|value| value.0.as_str())
        .ok_or_else(|| "CreateSession did not return session_handle".to_string())?;
    Path::new(handle_text.to_string())
        .map(Path::into_static)
        .map_err(|_| format!("invalid session_handle from portal: {handle_text}"))
}

fn select_monitor_source(conn: &SyncConnection, session: &Path<'static>) -> Result<(), String> {
    let mut options = PropMap::new();
    options.insert(
        "handle_token".to_string(),
        Variant(Box::new(new_token("select"))),
    );
    options.insert(
        "type".to_string(),
        Variant(Box::new(SCREENCAST_SOURCE_MONITOR)),
    );
    options.insert("multiple".to_string(), Variant(Box::new(false)));
    let _ = call_portal_request(conn, "SelectSources", (session.clone(), options))?;
    Ok(())
}

fn start_session(conn: &SyncConnection, session: &Path<'static>) -> Result<PropMap, String> {
    let mut options = PropMap::new();
    options.insert(
        "handle_token".to_string(),
        Variant(Box::new(new_token("start"))),
    );
    call_portal_request(conn, "Start", (session.clone(), String::new(), options))
}

fn open_pipewire_remote(conn: &SyncConnection, session: &Path<'static>) -> Result<OwnedFd, String> {
    let proxy = portal_proxy(conn);
    let options = PropMap::new();
    let (fd,): (OwnedFd,) = proxy
        .method_call(
            SCREENCAST_IFACE,
            "OpenPipeWireRemote",
            (session.clone(), options),
        )
        .map_err(|err| format!("OpenPipeWireRemote failed: {err}"))?;
    Ok(fd)
}

fn call_portal_request<A: AppendAll>(
    conn: &SyncConnection,
    method: &str,
    args: A,
) -> Result<PropMap, String> {
    let proxy = portal_proxy(conn);
    let (request_handle,): (Path<'static>,) = proxy
        .method_call(SCREENCAST_IFACE, method, args)
        .map_err(|err| format!("portal {method} call failed: {err}"))?;
    let response = wait_request_response(conn, request_handle)?;
    if response.status != 0 {
        return Err(format!(
            "portal {method} response status={}",
            response.status
        ));
    }
    Ok(response.results)
}

fn wait_request_response(
    conn: &SyncConnection,
    request_path: Path<'static>,
) -> Result<PortalRequestResponse, String> {
    let request_path_text = request_path.to_string();
    let response_slot: Arc<Mutex<Option<PortalRequestResponse>>> = Arc::new(Mutex::new(None));
    let response_slot_reader = Arc::clone(&response_slot);
    let mut rule = MatchRule::new_signal(
        PortalRequestResponse::INTERFACE,
        PortalRequestResponse::NAME,
    );
    rule.path = Some(request_path);
    let token = conn
        .add_match(rule, move |response: PortalRequestResponse, _conn, _msg| {
            if let Ok(mut slot) = response_slot_reader.lock() {
                *slot = Some(response);
            }
            false
        })
        .map_err(|err| format!("add request signal match failed: {err}"))?;

    let deadline = Instant::now() + REQUEST_WAIT_TIMEOUT;
    while Instant::now() < deadline {
        conn.process(Duration::from_millis(500))
            .map_err(|err| format!("wait request response failed: {err}"))?;
        if let Ok(mut slot) = response_slot.lock()
            && let Some(response) = slot.take()
        {
            let _ = conn.remove_match(token);
            return Ok(response);
        }
    }

    let _ = conn.remove_match(token);
    Err(format!(
        "timeout waiting for portal response at {request_path_text}"
    ))
}

fn parse_primary_stream(results: &PropMap) -> Result<StreamInfo, String> {
    let streams = results
        .get("streams")
        .ok_or_else(|| "Start response missing streams".to_string())?;
    let mut stream_items = streams
        .0
        .as_iter()
        .ok_or_else(|| "Start response streams has invalid type, expected a(ua{sv})".to_string())?;
    let primary = stream_items
        .next()
        .ok_or_else(|| "Start response streams is empty".to_string())?;
    parse_stream_tuple(primary)
}

fn parse_stream_tuple(stream: &dyn RefArg) -> Result<StreamInfo, String> {
    let mut stream_fields = stream
        .as_iter()
        .ok_or_else(|| "invalid stream entry: expected (ua{sv})".to_string())?;
    let node_id = stream_fields
        .next()
        .and_then(refarg_to_u32)
        .ok_or_else(|| "invalid stream entry: node id is missing".to_string())?;
    let props = stream_fields
        .next()
        .ok_or_else(|| "invalid stream entry: properties missing".to_string())?;
    Ok(StreamInfo {
        node_id,
        position: dict_lookup_i32_pair(props, "position"),
        size: dict_lookup_i32_pair(props, "size"),
    })
}

fn dict_lookup_i32_pair(dict_ref: &dyn RefArg, key: &str) -> Option<(i32, i32)> {
    let mut items = dict_ref.as_iter()?;
    loop {
        let item_key = items.next()?;
        let item_value = items.next()?;
        if item_key.as_str() == Some(key) {
            return refarg_to_i32_pair(unwrap_variant(item_value));
        }
    }
}

fn unwrap_variant(arg: &dyn RefArg) -> &dyn RefArg {
    if arg.arg_type() != ArgType::Variant {
        return arg;
    }
    let Some(mut value_iter) = arg.as_iter() else {
        return arg;
    };
    value_iter.next().unwrap_or(arg)
}

fn refarg_to_i32_pair(arg: &dyn RefArg) -> Option<(i32, i32)> {
    let mut values = arg.as_iter()?;
    let left = values.next().and_then(refarg_to_i32)?;
    let right = values.next().and_then(refarg_to_i32)?;
    Some((left, right))
}

fn refarg_to_i32(arg: &dyn RefArg) -> Option<i32> {
    if let Some(value) = arg.as_i64() {
        return i32::try_from(value).ok();
    }
    arg.as_u64().and_then(|value| i32::try_from(value).ok())
}

fn refarg_to_u32(arg: &dyn RefArg) -> Option<u32> {
    if let Some(value) = arg.as_u64() {
        return u32::try_from(value).ok();
    }
    arg.as_i64().and_then(|value| u32::try_from(value).ok())
}

fn portal_proxy(conn: &SyncConnection) -> Proxy<'_, &SyncConnection> {
    conn.with_proxy(PORTAL_DEST, PORTAL_PATH, DBUS_CALL_TIMEOUT)
}

fn new_token(prefix: &str) -> String {
    let millis = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|value| value.as_millis())
        .unwrap_or(0);
    format!("{prefix}-{}-{millis}", std::process::id())
}
