use std::path::PathBuf;
use std::process::Command;

use serde_json::Value;

use crate::sandbox::SandboxConfig;
use crate::sandbox::path_policy::{resolve_read_path, resolve_write_path};

use super::{
    DEFAULT_OUTPUT_CHAR_COUNT, DEFAULT_WAIT_DURATION_SECONDS, DEFAULT_WAIT_MS_BEFORE_ASYNC,
};

pub(super) struct StartParams {
    pub(super) operation: StartOperation,
    pub(super) prompt: String,
    pub(super) cwd: Option<PathBuf>,
    pub(super) output_path: Option<PathBuf>,
    pub(super) model: Option<String>,
    pub(super) full_auto: bool,
    pub(super) skip_git_repo_check: bool,
    pub(super) json_flag: bool,
    pub(super) wait_ms_before_async: u64,
    pub(super) output_char_count: usize,
}

pub(super) enum StartOperation {
    Start,
    Resume { session_id: String },
    Fork { session_id: String },
}

pub(super) struct StatusParams {
    pub(super) command_id: String,
    pub(super) session_id: String,
    pub(super) wait_duration_seconds: u64,
    pub(super) output_char_count: usize,
}

impl StartParams {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        let operation = StartOperation::parse(
            required_string(params, "op")?,
            optional_string(params, "session_id")?,
        )?;
        Ok(Self {
            operation,
            prompt: required_string(params, "prompt")?,
            cwd: resolve_cwd(optional_string(params, "cwd")?)?,
            output_path: resolve_output_path(optional_string(params, "output_path")?)?,
            model: optional_string(params, "model")?,
            full_auto: optional_bool(params, "full_auto")?.unwrap_or(true),
            skip_git_repo_check: optional_bool(params, "skip_git_repo_check")?.unwrap_or(true),
            json_flag: optional_bool(params, "json")?.unwrap_or(true),
            wait_ms_before_async: optional_u64(params, "wait_ms_before_async")?
                .unwrap_or(DEFAULT_WAIT_MS_BEFORE_ASYNC),
            output_char_count: optional_output_char_count(params, "output_character_count")?
                .unwrap_or(DEFAULT_OUTPUT_CHAR_COUNT),
        })
    }

    pub(super) fn output_path_value(&self) -> Option<String> {
        self.output_path
            .as_ref()
            .map(|path| path.to_string_lossy().to_string())
    }
}

impl StartOperation {
    fn parse(raw: String, session_id: Option<String>) -> Result<Self, String> {
        let op = raw.to_lowercase();
        match op.as_str() {
            "start" => Ok(Self::Start),
            "resume" => Ok(Self::Resume {
                session_id: require_session_id(session_id)?,
            }),
            "fork" => Ok(Self::Fork {
                session_id: require_session_id(session_id)?,
            }),
            _ => Err("op must be one of: start, resume, fork".to_string()),
        }
    }

    pub(super) fn apply(&self, command: &mut Command, prompt: &str) {
        match self {
            Self::Start => {
                command.arg("exec").arg(prompt);
            }
            Self::Resume { session_id } => {
                command
                    .arg("exec")
                    .arg("resume")
                    .arg("--session-id")
                    .arg(session_id)
                    .arg(prompt);
            }
            Self::Fork { session_id } => {
                command
                    .arg("fork")
                    .arg("--session-id")
                    .arg(session_id)
                    .arg(prompt);
            }
        }
    }
}

impl StatusParams {
    pub(super) fn parse(params: &Value) -> Result<Self, String> {
        let command_id = optional_string(params, "command_id")?.unwrap_or_default();
        let session_id = optional_string(params, "session_id")?.unwrap_or_default();
        if command_id.is_empty() && session_id.is_empty() {
            return Err("command_id or session_id is required".to_string());
        }
        Ok(Self {
            command_id,
            session_id,
            wait_duration_seconds: optional_u64(params, "wait_duration_seconds")?
                .unwrap_or(DEFAULT_WAIT_DURATION_SECONDS),
            output_char_count: optional_output_char_count(params, "output_character_count")?
                .unwrap_or(DEFAULT_OUTPUT_CHAR_COUNT),
        })
    }

    pub(super) fn has_command_id(&self) -> bool {
        !self.command_id.is_empty()
    }

    pub(super) fn has_session_id(&self) -> bool {
        !self.session_id.is_empty()
    }
}

fn resolve_cwd(raw: Option<String>) -> Result<Option<PathBuf>, String> {
    resolve_optional_path(raw, resolve_read_path)
}

fn resolve_output_path(raw: Option<String>) -> Result<Option<PathBuf>, String> {
    resolve_optional_path(raw, resolve_write_path)
}

fn resolve_optional_path(
    raw: Option<String>,
    resolver: fn(&str, &SandboxConfig) -> Result<PathBuf, String>,
) -> Result<Option<PathBuf>, String> {
    let Some(raw) = raw else {
        return Ok(None);
    };
    let config = SandboxConfig::default();
    resolver(&raw, &config).map(Some)
}

fn require_session_id(session_id: Option<String>) -> Result<String, String> {
    session_id.ok_or_else(|| "session_id is required for resume/fork".to_string())
}

fn required_string(params: &Value, field: &str) -> Result<String, String> {
    params
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .ok_or_else(|| format!("{field} is required"))
}

fn optional_string(params: &Value, field: &str) -> Result<Option<String>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    let value = raw
        .as_str()
        .ok_or_else(|| format!("{field} must be a string"))?
        .trim();
    if value.is_empty() {
        return Ok(None);
    }
    Ok(Some(value.to_string()))
}

fn optional_bool(params: &Value, field: &str) -> Result<Option<bool>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_bool()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a boolean"))
}

fn optional_u64(params: &Value, field: &str) -> Result<Option<u64>, String> {
    let Some(raw) = params.get(field) else {
        return Ok(None);
    };
    raw.as_u64()
        .map(Some)
        .ok_or_else(|| format!("{field} must be a non-negative integer"))
}

fn optional_output_char_count(params: &Value, field: &str) -> Result<Option<usize>, String> {
    let value = optional_u64(params, field)?;
    Ok(value.map(|count| count as usize))
}
