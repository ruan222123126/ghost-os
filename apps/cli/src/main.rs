// CLI entrypoint: parses commands and starts the interactive or one-shot execution flow.

mod client;
mod commands;
mod config;
mod envelope_generated;
mod repl;
mod types;

use anyhow::{Result, bail};
use clap::Parser;
use colored::Colorize;

use crate::client::BridgeClient;
use crate::config::Config;
use crate::repl::Repl;

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

fn main() {
    if let Err(err) = run() {
        eprintln!("{}", format!("Error: {err}").red());
        std::process::exit(1);
    }
}

fn run() -> Result<()> {
    let args = Args::parse();
    let config = Config::load(args.bridge_url, args.timeout)?;
    let client = BridgeClient::new(config)?;

    if let Some(message) = args.message {
        let text = message.trim();
        if text.is_empty() {
            bail!("--message cannot be empty");
        }
        let payload = client.send_message(text, None)?;
        if payload.status == "awaiting_human" {
            bail!("--message mode does not support ask_human interactions; use interactive mode");
        }
        println!("{}", payload.message);
        return Ok(());
    }

    client.health_check()?;

    let mut repl = Repl::new(client)?;
    repl.run()
}
