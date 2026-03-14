mod clear;
mod execute;
mod parse;
mod view;

use anyhow::Result;

use crate::client::BridgeClient;

pub use execute::CommandAction;
use execute::{CommandContext, execute_command};
use parse::parse_command;
use view::render_output;

pub fn handle_command(
    input: &str,
    client: &BridgeClient,
    current_session_id: &mut Option<String>,
) -> Result<CommandAction> {
    let command = parse_command(input)?;
    let outcome = execute_command(command, CommandContext::new(client, current_session_id))?;
    render_output(&outcome.output)?;
    Ok(outcome.action)
}
