use std::process::Child;
use std::thread;

use serde::{Deserialize, Serialize};

use crate::sandbox::SandboxConfig;

#[derive(Deserialize, Serialize)]
pub(super) struct ScriptWorkerRequest {
    pub(super) script: String,
    pub(super) max_memory_mb: u64,
    pub(super) sandbox_config: SandboxConfig,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct ScriptExecutionBudget {
    pub(super) timeout_ms: u64,
    pub(super) max_memory_mb: u64,
}

pub(super) struct WorkerProcess {
    pub(super) child: Child,
    pub(super) stdout_handle: thread::JoinHandle<Vec<u8>>,
    pub(super) stderr_handle: thread::JoinHandle<Vec<u8>>,
}
