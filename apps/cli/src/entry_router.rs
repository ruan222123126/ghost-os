use std::ffi::OsString;

use anyhow::{Result, bail};
use clap::Parser;

use crate::client::BridgeClient;
use crate::config::{CliConfigArgs, Config};
use crate::repl::Repl;
use crate::startup_profiler::{StartupCheckpoint, StartupProfiler};

#[derive(Parser, Debug)]
#[command(name = "ghost-cli", version, about = "Ghost-OS terminal client")]
struct Args {
    #[arg(long, value_name = "URL", help = "Bridge base URL")]
    bridge_url: Option<String>,
    #[arg(
        long,
        value_name = "SECONDS",
        help = "Request timeout in seconds (overrides GHOST_CLI_TIMEOUT)"
    )]
    timeout: Option<u64>,
    #[arg(short, long, value_name = "TEXT", help = "Send one message and exit")]
    message: Option<String>,
}

struct RouteConfig {
    bridge_url: Option<String>,
    timeout: Option<u64>,
}

impl RouteConfig {
    fn new(bridge_url: Option<String>, timeout: Option<u64>) -> Self {
        Self {
            bridge_url,
            timeout,
        }
    }
}

pub fn run(profiler: &mut StartupProfiler) -> Result<()> {
    if has_version_shortcut(std::env::args_os()) {
        print_version();
        return Ok(());
    }

    let args = Args::parse();
    profiler.checkpoint(StartupCheckpoint::ArgsParsed);
    route(args, profiler)
}

fn route(args: Args, profiler: &mut StartupProfiler) -> Result<()> {
    let Args {
        bridge_url,
        timeout,
        message,
    } = args;

    let route_config = RouteConfig::new(bridge_url, timeout);
    if let Some(message) = message {
        return handle_message_mode(route_config, message, profiler);
    }

    handle_repl_mode(route_config, profiler)
}

fn handle_message_mode(
    route_config: RouteConfig,
    message: String,
    profiler: &mut StartupProfiler,
) -> Result<()> {
    let text = message.trim();
    if text.is_empty() {
        bail!("--message cannot be empty");
    }

    let client = init_client(route_config, profiler)?;
    let payload = client.send_message(text, None)?;
    if payload.as_awaiting_human().is_some() {
        bail!("--message mode does not support ask_human interactions; use interactive mode");
    }
    if let Some(reply) = payload.into_success() {
        println!("{}", reply.message);
    }

    profiler.checkpoint(StartupCheckpoint::OneshotDone);
    Ok(())
}

fn handle_repl_mode(route_config: RouteConfig, profiler: &mut StartupProfiler) -> Result<()> {
    let client = init_client(route_config, profiler)?;
    client.health_check()?;

    let mut repl = Repl::new(client)?;
    profiler.checkpoint(StartupCheckpoint::ReplStarted);
    repl.run()
}

fn init_client(route_config: RouteConfig, profiler: &mut StartupProfiler) -> Result<BridgeClient> {
    let config = Config::load(CliConfigArgs::new(
        route_config.bridge_url,
        route_config.timeout,
    ))?;
    profiler.checkpoint(StartupCheckpoint::ConfigLoaded);

    let client = BridgeClient::new(config)?;
    profiler.checkpoint(StartupCheckpoint::ClientReady);
    Ok(client)
}

fn has_version_shortcut<I>(raw_args: I) -> bool
where
    I: IntoIterator<Item = OsString>,
{
    for arg in raw_args.into_iter().skip(1) {
        let Some(value) = arg.to_str() else {
            continue;
        };
        if value == "--" {
            break;
        }
        if matches!(value, "--version" | "-v" | "-V") {
            return true;
        }
    }

    false
}

fn print_version() {
    println!("{} {}", env!("CARGO_PKG_NAME"), env!("CARGO_PKG_VERSION"));
}

#[cfg(test)]
mod tests {
    use super::has_version_shortcut;
    use std::ffi::OsString;

    #[test]
    fn version_shortcut_matches_supported_flags() {
        assert!(has_version_shortcut(argv(["ghost-cli", "--version"])));
        assert!(has_version_shortcut(argv(["ghost-cli", "-V"])));
        assert!(has_version_shortcut(argv(["ghost-cli", "-v"])));
    }

    #[test]
    fn version_shortcut_ignores_flags_after_separator() {
        assert!(!has_version_shortcut(argv(["ghost-cli", "--", "-v"])));
    }

    #[test]
    fn version_shortcut_ignores_regular_invocation() {
        assert!(!has_version_shortcut(argv([
            "ghost-cli",
            "--message",
            "hello"
        ])));
    }

    fn argv<const N: usize>(items: [&str; N]) -> Vec<OsString> {
        items.into_iter().map(OsString::from).collect()
    }
}
