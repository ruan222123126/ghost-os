use std::process::Child;

use super::UNKNOWN_EXIT_CODE;
use super::output::{SharedOutputBuffer, SharedSessionId, output_tail, session_id_value};

pub(super) struct CodexCommand {
    seq: u64,
    id: String,
    child: Child,
    output: SharedOutputBuffer,
    session_id: SharedSessionId,
    output_path: Option<String>,
    exit_code: Option<i32>,
}

impl CodexCommand {
    pub(super) fn new(
        seq: u64,
        id: String,
        child: Child,
        output: SharedOutputBuffer,
        session_id: SharedSessionId,
        output_path: Option<String>,
    ) -> Self {
        Self {
            seq,
            id,
            child,
            output,
            session_id,
            output_path,
            exit_code: None,
        }
    }

    pub(super) fn id(&self) -> &str {
        &self.id
    }

    pub(super) fn seq(&self) -> u64 {
        self.seq
    }

    pub(super) fn session_id_value(&self) -> Option<String> {
        session_id_value(&self.session_id)
    }

    pub(super) fn output_tail(&self, max_chars: usize) -> String {
        output_tail(&self.output, max_chars)
    }

    pub(super) fn output_path_value(&self) -> String {
        self.output_path.clone().unwrap_or_default()
    }

    pub(super) fn update_exit_code(&mut self) -> Result<Option<i32>, String> {
        if let Some(code) = self.exit_code {
            return Ok(Some(code));
        }
        match self.child.try_wait() {
            Ok(Some(status)) => {
                let code = status.code().unwrap_or(UNKNOWN_EXIT_CODE);
                self.exit_code = Some(code);
                Ok(Some(code))
            }
            Ok(None) => Ok(None),
            Err(err) => Err(format!("check process status failed: {err}")),
        }
    }
}
