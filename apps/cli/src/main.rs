// CLI entrypoint shell. Detailed startup routing lives in `entry_router`.

mod client;
mod commands;
mod config;
mod entry_router;
#[allow(dead_code)]
mod envelope_generated;
mod repl;
mod startup_profiler;
mod types;

use anyhow::Result;
use colored::Colorize;

use crate::startup_profiler::{StartupCheckpoint, StartupProfiler};

fn main() {
    if let Err(err) = run() {
        eprintln!("{}", format!("Error: {err}").red());
        std::process::exit(1);
    }
}

fn run() -> Result<()> {
    let mut profiler = StartupProfiler::from_env()?;
    profiler.checkpoint(StartupCheckpoint::Entry);

    let run_result = entry_router::run(&mut profiler);
    merge_results(run_result, profiler.finish())
}

fn merge_results(run_result: Result<()>, profile_result: Result<()>) -> Result<()> {
    match (run_result, profile_result) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(err), Ok(())) => Err(err),
        (Ok(()), Err(profile_err)) => Err(profile_err),
        (Err(err), Err(profile_err)) => {
            Err(err.context(format!("startup profiler flush failed: {profile_err}")))
        }
    }
}
