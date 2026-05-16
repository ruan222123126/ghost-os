mod actions;
mod executable;
mod launch_spec;
mod path_env;
mod worker;

use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::Response;

pub(crate) const CODEX_CLI_STATUS_POLL_INTERVAL_MS: u64 = 500;
pub(crate) const CODEX_CLI_UNKNOWN_EXIT_CODE: i32 = -1;
pub(crate) const DEFAULT_WAIT_MS_BEFORE_ASYNC: usize = 3000;
pub(crate) const DEFAULT_WAIT_DURATION_SECONDS: usize = 300;
pub(crate) const DEFAULT_OUTPUT_CHARACTER_COUNT: usize = 200;

#[derive(Debug)]
pub(crate) struct StartRequest {
    pub(crate) op: String,
    pub(crate) prompt: String,
    pub(crate) session_id: Option<String>,
    pub(crate) working_dir: String,
    pub(crate) use_cwd_flag: bool,
    pub(crate) output_path: Option<String>,
    pub(crate) model: Option<String>,
    pub(crate) codex_executable_path: Option<String>,
    pub(crate) node_executable_path: Option<String>,
    pub(crate) sandbox: Option<String>,
    pub(crate) full_auto: Option<bool>,
    pub(crate) skip_git_repo_check: bool,
    pub(crate) json_flag: bool,
    pub(crate) wait_ms_before_async: usize,
    pub(crate) output_character_count: usize,
}

#[derive(Debug)]
pub(crate) struct StatusRequest {
    pub(crate) output_path: String,
    pub(crate) exit_code_path: String,
    pub(crate) wait_duration_seconds: usize,
    pub(crate) output_character_count: usize,
}

#[derive(Debug, Deserialize, Serialize)]
pub(crate) struct WorkerRequest {
    pub(crate) args: Vec<String>,
    pub(crate) working_dir: String,
    pub(crate) output_path: String,
    pub(crate) exit_code_path: String,
    pub(crate) codex_executable_path: Option<String>,
    pub(crate) node_executable_path: Option<String>,
}

pub(crate) fn dispatch_action(action: &str, params: &Value) -> Option<Response> {
    actions::dispatch_action(action, params)
}

pub(crate) fn run_worker() {
    worker::run_worker()
}
