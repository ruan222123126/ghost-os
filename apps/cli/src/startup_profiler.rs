use std::env;
use std::fs::File;
use std::io::{self, Write};
use std::path::PathBuf;
use std::time::{Duration, Instant};

use anyhow::{Context, Result, bail};

const ENV_PROFILE_STARTUP: &str = "GHOST_CLI_PROFILE_STARTUP";
const ENV_PROFILE_STARTUP_FILE: &str = "GHOST_CLI_PROFILE_STARTUP_FILE";

#[derive(Debug, Clone, Copy)]
pub enum StartupCheckpoint {
    Entry,
    ArgsParsed,
    ConfigLoaded,
    ClientReady,
    ReplStarted,
    OneshotDone,
}

impl StartupCheckpoint {
    fn label(self) -> &'static str {
        match self {
            Self::Entry => "entry",
            Self::ArgsParsed => "args_parsed",
            Self::ConfigLoaded => "config_loaded",
            Self::ClientReady => "client_ready",
            Self::ReplStarted => "repl_started",
            Self::OneshotDone => "oneshot_done",
        }
    }
}

enum ProfileOutput {
    Off,
    Stderr,
    File(PathBuf),
}

#[derive(Debug, Clone)]
struct CheckpointRecord {
    name: &'static str,
    since_last: Duration,
    since_start: Duration,
}

pub struct StartupProfiler {
    output: ProfileOutput,
    start: Instant,
    last: Instant,
    checkpoints: Vec<CheckpointRecord>,
}

impl StartupProfiler {
    pub fn from_env() -> Result<Self> {
        let output = resolve_output_from_env()?;
        Ok(Self::new(output))
    }

    fn new(output: ProfileOutput) -> Self {
        let now = Instant::now();
        Self {
            output,
            start: now,
            last: now,
            checkpoints: Vec::new(),
        }
    }

    pub fn checkpoint(&mut self, checkpoint: StartupCheckpoint) {
        if matches!(self.output, ProfileOutput::Off) {
            return;
        }

        let now = Instant::now();
        self.checkpoints.push(CheckpointRecord {
            name: checkpoint.label(),
            since_last: now.duration_since(self.last),
            since_start: now.duration_since(self.start),
        });
        self.last = now;
    }

    pub fn finish(&self) -> Result<()> {
        if matches!(self.output, ProfileOutput::Off) {
            return Ok(());
        }

        let report = render_report(&self.checkpoints, self.start.elapsed());
        self.write_report(&report)
    }

    fn write_report(&self, report: &str) -> Result<()> {
        match &self.output {
            ProfileOutput::Off => Ok(()),
            ProfileOutput::Stderr => write_to_stderr(report),
            ProfileOutput::File(path) => write_to_file(path, report),
        }
    }
}

fn resolve_output_from_env() -> Result<ProfileOutput> {
    let Some(raw) = env::var_os(ENV_PROFILE_STARTUP) else {
        return Ok(ProfileOutput::Off);
    };

    let value = raw.to_string_lossy();
    let value = value.trim();
    if value.is_empty() || value == "0" {
        return Ok(ProfileOutput::Off);
    }
    if value != "1" {
        bail!("{ENV_PROFILE_STARTUP} must be 1 to enable startup profiling, got {value:?}");
    }

    match env::var(ENV_PROFILE_STARTUP_FILE) {
        Ok(path) => {
            let path = path.trim();
            if path.is_empty() {
                bail!("{ENV_PROFILE_STARTUP_FILE} cannot be empty when profiling is enabled");
            }
            Ok(ProfileOutput::File(PathBuf::from(path)))
        }
        Err(env::VarError::NotPresent) => Ok(ProfileOutput::Stderr),
        Err(env::VarError::NotUnicode(_)) => {
            bail!("{ENV_PROFILE_STARTUP_FILE} must be valid UTF-8")
        }
    }
}

fn write_to_stderr(report: &str) -> Result<()> {
    let mut stderr = io::stderr().lock();
    stderr
        .write_all(report.as_bytes())
        .context("failed to write startup profile to stderr")?;
    stderr
        .flush()
        .context("failed to flush startup profile to stderr")
}

fn write_to_file(path: &PathBuf, report: &str) -> Result<()> {
    let mut file = File::create(path)
        .with_context(|| format!("failed to create startup profile file {}", path.display()))?;
    file.write_all(report.as_bytes())
        .with_context(|| format!("failed to write startup profile file {}", path.display()))?;
    file.flush()
        .with_context(|| format!("failed to flush startup profile file {}", path.display()))
}

fn render_report(checkpoints: &[CheckpointRecord], total: Duration) -> String {
    let mut lines = Vec::with_capacity(checkpoints.len() + 2);
    lines.push("ghost-cli startup profile".to_string());
    for checkpoint in checkpoints {
        lines.push(format!(
            "{:<14} +{:>10} total {:>10}",
            checkpoint.name,
            format_duration(checkpoint.since_last),
            format_duration(checkpoint.since_start)
        ));
    }
    lines.push(format!("{:<14} {:>10}", "total", format_duration(total)));
    format!("{}\n", lines.join("\n"))
}

fn format_duration(duration: Duration) -> String {
    format!("{:.3}ms", duration.as_secs_f64() * 1000.0)
}

#[cfg(test)]
mod tests {
    use super::{CheckpointRecord, format_duration, render_report};
    use std::time::Duration;

    #[test]
    fn format_duration_uses_milliseconds() {
        let text = format_duration(Duration::from_micros(1500));
        assert_eq!(text, "1.500ms");
    }

    #[test]
    fn render_report_includes_total_and_checkpoints() {
        let checkpoints = vec![CheckpointRecord {
            name: "entry",
            since_last: Duration::from_millis(1),
            since_start: Duration::from_millis(1),
        }];

        let output = render_report(&checkpoints, Duration::from_millis(5));
        assert!(output.contains("ghost-cli startup profile"));
        assert!(output.contains("entry"));
        assert!(output.contains("total"));
    }
}
