use dbus::arg::{PropMap, ReadAll};
use dbus::message::SignalArgs;

const REQUEST_IFACE: &str = "org.freedesktop.portal.Request";
const REQUEST_SIGNAL_NAME: &str = "Response";

#[derive(Debug)]
pub(crate) struct PortalRequestResponse {
    pub(crate) status: u32,
    pub(crate) results: PropMap,
}

impl dbus::arg::AppendAll for PortalRequestResponse {
    fn append(&self, i: &mut dbus::arg::IterAppend) {
        dbus::arg::RefArg::append(&self.status, i);
        dbus::arg::RefArg::append(&self.results, i);
    }
}

impl ReadAll for PortalRequestResponse {
    fn read(i: &mut dbus::arg::Iter) -> Result<Self, dbus::arg::TypeMismatchError> {
        Ok(Self {
            status: i.read()?,
            results: i.read()?,
        })
    }
}

impl SignalArgs for PortalRequestResponse {
    const NAME: &'static str = REQUEST_SIGNAL_NAME;
    const INTERFACE: &'static str = REQUEST_IFACE;
}
