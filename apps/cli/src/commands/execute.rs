use anyhow::Result;

use crate::client::BridgeClient;
use crate::types::{BridgeConfig, ConfigUpdate};

use super::clear::clear_screen;
use super::parse::{CliCommand, ConfigMutation, ResettableValue};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CommandAction {
    Continue,
    Exit,
}

pub(crate) struct CommandContext<'a> {
    client: &'a BridgeClient,
    current_session_id: &'a mut Option<String>,
}

impl<'a> CommandContext<'a> {
    pub(crate) fn new(
        client: &'a BridgeClient,
        current_session_id: &'a mut Option<String>,
    ) -> Self {
        Self {
            client,
            current_session_id,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub(crate) struct CommandExecution {
    pub(crate) action: CommandAction,
    pub(crate) output: CommandOutput,
}

impl CommandExecution {
    fn continue_with(output: CommandOutput) -> Self {
        Self {
            action: CommandAction::Continue,
            output,
        }
    }

    fn exit() -> Self {
        Self {
            action: CommandAction::Exit,
            output: CommandOutput::None,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub(crate) enum CommandOutput {
    None,
    Help,
    ConfigSnapshot(BridgeConfig),
    SessionStatus(SessionStatus),
    SessionCleared,
    ConfigUpdated(ConfigUpdateFeedback),
    LiteralHint,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum SessionStatus {
    Active(String),
    Inactive,
}

#[derive(Debug, Clone, PartialEq)]
pub(crate) struct ConfigUpdateFeedback {
    pub(crate) kind: ConfigChangeKind,
    pub(crate) config: BridgeConfig,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum ConfigChangeKind {
    Model,
    Provider,
    BaseUrlSet,
    BaseUrlReset,
    ChatPathSet,
    ChatPathReset,
    ApiKeyUpdated,
    ApiKeyCleared,
}

pub(crate) fn execute_command(
    command: CliCommand,
    context: CommandContext<'_>,
) -> Result<CommandExecution> {
    match command {
        CliCommand::Help => Ok(CommandExecution::continue_with(CommandOutput::Help)),
        CliCommand::Exit => Ok(CommandExecution::exit()),
        CliCommand::Clear => execute_clear(),
        CliCommand::Config => execute_show_config(context.client),
        CliCommand::Session => Ok(show_session(context.current_session_id.as_deref())),
        CliCommand::NewSession => Ok(clear_session(context.current_session_id)),
        CliCommand::UpdateConfig(change) => execute_config_update(context.client, change),
        CliCommand::LiteralHint => Ok(CommandExecution::continue_with(CommandOutput::LiteralHint)),
    }
}

fn execute_clear() -> Result<CommandExecution> {
    clear_screen()?;
    Ok(CommandExecution::continue_with(CommandOutput::None))
}

fn execute_show_config(client: &BridgeClient) -> Result<CommandExecution> {
    let config = client.get_config()?;
    Ok(CommandExecution::continue_with(
        CommandOutput::ConfigSnapshot(config),
    ))
}

fn show_session(current_session_id: Option<&str>) -> CommandExecution {
    let output = match current_session_id {
        Some(session_id) if !session_id.trim().is_empty() => {
            CommandOutput::SessionStatus(SessionStatus::Active(session_id.to_string()))
        }
        _ => CommandOutput::SessionStatus(SessionStatus::Inactive),
    };
    CommandExecution::continue_with(output)
}

fn clear_session(current_session_id: &mut Option<String>) -> CommandExecution {
    *current_session_id = None;
    CommandExecution::continue_with(CommandOutput::SessionCleared)
}

fn execute_config_update(
    client: &BridgeClient,
    change: ConfigMutation,
) -> Result<CommandExecution> {
    let (update, kind) = build_config_update(change);
    let config = client.update_config(&update)?;
    let feedback = ConfigUpdateFeedback { kind, config };
    Ok(CommandExecution::continue_with(
        CommandOutput::ConfigUpdated(feedback),
    ))
}

fn build_config_update(change: ConfigMutation) -> (ConfigUpdate, ConfigChangeKind) {
    match change {
        ConfigMutation::Model(model) => (
            ConfigUpdate {
                model: Some(model),
                ..ConfigUpdate::empty()
            },
            ConfigChangeKind::Model,
        ),
        ConfigMutation::Provider(provider) => (
            ConfigUpdate {
                provider: Some(provider),
                ..ConfigUpdate::empty()
            },
            ConfigChangeKind::Provider,
        ),
        ConfigMutation::BaseUrl(value) => resettable_update(value, UpdateTarget::BaseUrl),
        ConfigMutation::ChatPath(value) => resettable_update(value, UpdateTarget::ChatPath),
        ConfigMutation::ApiKey(api_key) => (
            ConfigUpdate {
                api_key: Some(api_key),
                ..ConfigUpdate::empty()
            },
            ConfigChangeKind::ApiKeyUpdated,
        ),
        ConfigMutation::ClearApiKey => (
            ConfigUpdate {
                api_key: Some(String::new()),
                ..ConfigUpdate::empty()
            },
            ConfigChangeKind::ApiKeyCleared,
        ),
    }
}

fn resettable_update(
    value: ResettableValue,
    target: UpdateTarget,
) -> (ConfigUpdate, ConfigChangeKind) {
    let (value, reset) = match value {
        ResettableValue::Value(value) => (value, false),
        ResettableValue::Default => (String::new(), true),
    };

    match target {
        UpdateTarget::BaseUrl => (
            ConfigUpdate {
                base_url: Some(value),
                ..ConfigUpdate::empty()
            },
            if reset {
                ConfigChangeKind::BaseUrlReset
            } else {
                ConfigChangeKind::BaseUrlSet
            },
        ),
        UpdateTarget::ChatPath => (
            ConfigUpdate {
                chat_path: Some(value),
                ..ConfigUpdate::empty()
            },
            if reset {
                ConfigChangeKind::ChatPathReset
            } else {
                ConfigChangeKind::ChatPathSet
            },
        ),
    }
}

enum UpdateTarget {
    BaseUrl,
    ChatPath,
}

#[cfg(test)]
#[path = "execute_tests.rs"]
mod tests;
